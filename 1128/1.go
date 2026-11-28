/*写一个 Storage 接口，包含 Save(key, value string) error 和 Get(key string) (string, error)。
实现一个内存版 MemoryStorage，
再写一个 UserService 通过构造函数注入 Storage。
给第1题写一个 FakeStorage，在测试里验证 UserService 保存用户后能读回来。*/
package main
import (
	"errors"
	"fmt"
)
type Storage interface {
	Save(key, value string) error
	Get(key string) (string error)

}
type MemoryStorage struct {data map[string]string}
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string](string)),
	}
}
func (m *MemoryStorage) Save(key, value string) error {
	m.data[key] = value
	return nil
}
func (m *MemoryStorage) Get(key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}
type UserService struct {
	store Storage
}
func NewUserService(store Storage) *UserService {
	return &UserService{
		store: store,
	}
}