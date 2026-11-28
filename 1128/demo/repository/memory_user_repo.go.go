package repository
import (
	"sync"
)
type User struct {
	ID int
	name string
}
type MemoryUserRepo struct {
	mu sync.Mutex
	users map[int]User
	next int
}

func NewMemoryUserRepo() *MemoryUserRepo {
	return &MemoryUserRepo{users: make(map[int](User))}
}
func (m *MemoryUserRepo) Save(u User) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	u.ID = m.next
	m.users[u.ID] = u
	return u.ID, nil
}
func (m *MemoryUserRepo) FindbyID(id int) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return User{}, ErrorNotFound
	}
	return u, nil
}
type NotFoundError struct{}
func (e *NotFoundError) Error() string {return "用户不存在"}
var ErrorNotFound = &NotFoundError{}