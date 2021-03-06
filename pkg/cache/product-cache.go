package cache

import "github.com/alkmc/restClean/pkg/entity"

//Cache is responsible for caching mechanism
type Cache interface {
	Set(key string, value *entity.Product)
	Get(key string) *entity.Product
	Expire(key string)
}
