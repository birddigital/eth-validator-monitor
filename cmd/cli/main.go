package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "eth-validator-monitor",
		Short: "CLI for managing Ethereum validator monitoring",
		Long:  `Command-line interface for adding validators, viewing stats, and managing the monitoring system.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add validator command
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a validator to monitor",
		Long:  `Add a validator by public key or index to the monitoring system.`,
		Run:   runAdd,
	}
	addCmd.Flags().String("pubkey", "", "Validator public key (0x...)")
	addCmd.Flags().Uint64("index", 0, "Validator index")
	addCmd.Flags().String("name", "", "Optional validator name")

	// List validators command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List monitored validators",
		Run:   runList,
	}
	listCmd.Flags().Int("limit", 50, "Maximum number of validators to show")
	listCmd.Flags().Int("offset", 0, "Offset for pagination")

	// Stats command
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show validator statistics",
		Run:   runStats,
	}
	statsCmd.Flags().Uint64("index", 0, "Validator index (required)")
	statsCmd.Flags().Int("days", 7, "Number of days of history")

	// Health check command
	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check system health",
		Run:   runHealth,
	}

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(healthCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAdd(cmd *cobra.Command, args []string) {
	pubkey, _ := cmd.Flags().GetString("pubkey")
	index, _ := cmd.Flags().GetUint64("index")
	name, _ := cmd.Flags().GetString("name")

	if pubkey == "" && index == 0 {
		fmt.Fprintf(os.Stderr, "Error: Must specify either --pubkey or --index\n")
		os.Exit(1)
	}

	pool := initDB()
	defer pool.Close()

	repo := repository.NewValidatorRepository(pool)
	ctx := context.Background()

	validator := &models.Validator{
		Pubkey:    pubkey,
		Monitored: true,
	}

	if name != "" {
		validator.Name = &name
	}

	if index > 0 {
		validator.ValidatorIndex = int64(index)
	}

	if err := repo.CreateValidator(ctx, validator); err != nil {
		log.Fatalf("Failed to add validator: %v", err)
	}

	fmt.Printf("✓ Validator added successfully\n")
	if validator.Pubkey != "" {
		fmt.Printf("  Public Key: %s\n", validator.Pubkey)
	}
	if validator.ValidatorIndex > 0 {
		fmt.Printf("  Index: %d\n", validator.ValidatorIndex)
	}
	if validator.Name != nil && *validator.Name != "" {
		fmt.Printf("  Name: %s\n", *validator.Name)
	}
}

func runList(cmd *cobra.Command, args []string) {
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	pool := initDB()
	defer pool.Close()

	repo := repository.NewValidatorRepository(pool)
	ctx := context.Background()

	filter := &models.ValidatorFilter{
		Limit:  limit,
		Offset: offset,
	}

	validators, err := repo.ListValidators(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to list validators: %v", err)
	}

	if len(validators) == 0 {
		fmt.Println("No validators found")
		return
	}

	fmt.Printf("Found %d validators:\n\n", len(validators))
	fmt.Printf("%-10s %-12s %-66s %s\n", "INDEX", "MONITORED", "PUBLIC KEY", "NAME")
	fmt.Println(strings.Repeat("-", 120))

	for _, v := range validators {
		pubkeyShort := v.Pubkey
		if len(pubkeyShort) > 20 {
			pubkeyShort = pubkeyShort[:18] + "..."
		}

		monitored := "yes"
		if !v.Monitored {
			monitored = "no"
		}

		name := ""
		if v.Name != nil {
			name = *v.Name
		}

		fmt.Printf("%-10d %-12s %-66s %s\n", v.ValidatorIndex, monitored, pubkeyShort, name)
	}
}

func runStats(cmd *cobra.Command, args []string) {
	index, _ := cmd.Flags().GetUint64("index")
	days, _ := cmd.Flags().GetInt("days")

	if index == 0 {
		fmt.Fprintf(os.Stderr, "Error: --index is required\n")
		os.Exit(1)
	}

	pool := initDB()
	defer pool.Close()

	repo := repository.NewSnapshotRepository(pool)
	ctx := context.Background()

	// Get latest snapshot
	latest, err := repo.GetLatestSnapshot(ctx, int64(index))
	if err != nil {
		log.Fatalf("Failed to get latest snapshot: %v", err)
	}

	if latest == nil {
		fmt.Printf("No data found for validator %d\n", index)
		return
	}

	fmt.Printf("Validator %d Statistics\n", index)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Latest Update: %s\n", latest.Time.Format("2006-01-02 15:04:05"))
	fmt.Printf("Balance: %.4f ETH\n", float64(latest.Balance)/1e9)
	fmt.Printf("Effective Balance: %.4f ETH\n", float64(latest.EffectiveBalance)/1e9)

	if latest.AttestationEffectiveness != nil {
		fmt.Printf("Effectiveness: %.2f%%\n", *latest.AttestationEffectiveness)
	}

	// Get recent snapshots
	recent, err := repo.GetRecentSnapshots(ctx, int64(index), days*24*5) // ~5 snapshots per hour
	if err != nil {
		log.Fatalf("Failed to get recent snapshots: %v", err)
	}

	if len(recent) > 0 {
		fmt.Printf("\nRecent Activity (%d snapshots):\n", len(recent))
		fmt.Printf("  First: %s\n", recent[len(recent)-1].Time.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Last:  %s\n", recent[0].Time.Format("2006-01-02 15:04:05"))
	}
}

func runHealth(cmd *cobra.Command, args []string) {
	pool := initDB()
	defer pool.Close()

	ctx := context.Background()

	// Test database connection
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("❌ Database: UNHEALTHY (%v)\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Database: HEALTHY")
	if cfgFile != "" {
		fmt.Printf("✓ Config loaded from: %s\n", cfgFile)
	}
	fmt.Println("\n✓ System is healthy")
}

func initDB() *pgxpool.Pool {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()
	poolConfig, err := cfg.Database.BuildPoolConfig()
	if err != nil {
		log.Fatalf("Failed to build pool config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return pool
}
