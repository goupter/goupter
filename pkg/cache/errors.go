package cache

import "errors"

// ErrNotFound 缓存未找到
var ErrNotFound = errors.New("cache: key not found")

// IsNotFound 检查是否为未找到错误
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return true
	}
	if errors.Is(err, ErrNotFound) {
		return true
	}
	if err.Error() == "redis: nil" {
		return true
	}
	return false
}
