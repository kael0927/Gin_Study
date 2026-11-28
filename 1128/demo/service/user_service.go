package service
import "demo/repository"

type UserRepository interface {
	Save(u repository.User) (id int, err error)
	FindByID(id int) (u repository.User, err error)
}
type UserService struct {
	repo UserRepository
}
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}
func (u *UserService) Register(name string) (int, error) {
	if name == "" {
		return 0, ErrEmptyName
	}
	return u.repo.Save(repository.User{Name: name})
}
func (u *UserService) GetUser(id int) (repository.User, error) {
	return u.repo.FindByID(id)
}
var ErrEmptyName = &EmptyNameError{}
type EmptyNameError struct{}
func (e *EmptyNameError) Error() string {
	return "用户名不能为空"
}