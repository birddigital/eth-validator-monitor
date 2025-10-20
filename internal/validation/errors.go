package validation

import (
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// GraphQLValidationError creates a GraphQL error with validation details
func GraphQLValidationError(err error) *gqlerror.Error {
	return &gqlerror.Error{
		Message: err.Error(),
		Extensions: map[string]interface{}{
			"code": "VALIDATION_ERROR",
		},
	}
}

// GraphQLAuthError creates a GraphQL error for authentication failures
func GraphQLAuthError(message string) *gqlerror.Error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code": "UNAUTHORIZED",
		},
	}
}

// GraphQLNotFoundError creates a GraphQL error for resource not found
func GraphQLNotFoundError(resource string) *gqlerror.Error {
	return &gqlerror.Error{
		Message: resource + " not found",
		Extensions: map[string]interface{}{
			"code": "NOT_FOUND",
		},
	}
}

// GraphQLInternalError creates a GraphQL error for internal errors
func GraphQLInternalError(message string) *gqlerror.Error {
	return &gqlerror.Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code": "INTERNAL_ERROR",
		},
	}
}
