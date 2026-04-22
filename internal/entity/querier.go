package entity

import (
	"context"
)

type Querier interface {
	GetUserByUsername(ctx context.Context, username string) (User, error)
}
