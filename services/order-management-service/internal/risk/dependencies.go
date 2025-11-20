package risk

import (
	"github.com/redis/go-redis/v9"

	"github.com/jjongkwann/aipx/services/order-management-service/internal/repository"
)

// Dependencies holds the external dependencies required by risk rules
type Dependencies struct {
	Repository  repository.OrderRepository
	RedisClient *redis.Client
}
