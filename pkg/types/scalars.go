package types

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// Time is a custom scalar for time.Time
type Time time.Time

// MarshalGQL implements the graphql.Marshaler interface
func (t Time) MarshalGQL(w io.Writer) {
	tt := time.Time(t)
	w.Write([]byte(strconv.Quote(tt.Format(time.RFC3339))))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (t *Time) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("time must be a string")
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = Time(parsed)
	return nil
}

// MarshalJSON implements json.Marshaler
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t))
}

// UnmarshalJSON implements json.Unmarshaler
func (t *Time) UnmarshalJSON(data []byte) error {
	var tt time.Time
	if err := json.Unmarshal(data, &tt); err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

// ToTime converts Time to time.Time
func (t Time) ToTime() time.Time {
	return time.Time(t)
}

// BigInt is a custom scalar for big.Int
type BigInt big.Int

// MarshalGQL implements the graphql.Marshaler interface
func (b BigInt) MarshalGQL(w io.Writer) {
	bi := big.Int(b)
	graphql.MarshalString(bi.String()).MarshalGQL(w)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (b *BigInt) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("bigint must be a string")
	}

	bi := new(big.Int)
	_, success := bi.SetString(str, 10)
	if !success {
		return fmt.Errorf("failed to parse bigint: %s", str)
	}

	*b = BigInt(*bi)
	return nil
}

// MarshalJSON implements json.Marshaler
func (b BigInt) MarshalJSON() ([]byte, error) {
	bi := big.Int(b)
	return json.Marshal(bi.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (b *BigInt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	bi := new(big.Int)
	_, success := bi.SetString(str, 10)
	if !success {
		return fmt.Errorf("failed to parse bigint: %s", str)
	}

	*b = BigInt(*bi)
	return nil
}

// ToBigInt converts BigInt to *big.Int
func (b *BigInt) ToBigInt() *big.Int {
	bi := big.Int(*b)
	return &bi
}

// NewBigIntFromInt64 creates a BigInt from int64
func NewBigIntFromInt64(i int64) BigInt {
	return BigInt(*big.NewInt(i))
}

// NewBigIntFromBigInt creates a BigInt from *big.Int
func NewBigIntFromBigInt(bi *big.Int) BigInt {
	if bi == nil {
		return BigInt(*big.NewInt(0))
	}
	return BigInt(*bi)
}

// Hash is a custom scalar for Ethereum hashes (hex strings)
type Hash string

// MarshalGQL implements the graphql.Marshaler interface
func (h Hash) MarshalGQL(w io.Writer) {
	graphql.MarshalString(string(h)).MarshalGQL(w)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (h *Hash) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("hash must be a string")
	}

	// Validate hex format (0x prefixed)
	if len(str) < 2 || str[0:2] != "0x" {
		return fmt.Errorf("hash must be a hex string with 0x prefix")
	}

	// Validate hex characters
	for i := 2; i < len(str); i++ {
		c := str[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("hash contains invalid hex character: %c", c)
		}
	}

	*h = Hash(str)
	return nil
}

// MarshalJSON implements json.Marshaler
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(h))
}

// UnmarshalJSON implements json.Unmarshaler
func (h *Hash) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	return h.UnmarshalGQL(str)
}

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

// Bytes returns the hash as bytes (without 0x prefix)
func (h Hash) Bytes() []byte {
	if len(h) < 2 {
		return []byte{}
	}
	return []byte(h[2:])
}
