package logic

import(
	"bubble/dao"
	"bubble/models"
)
func DeleteTodo(id int) error {
	var todo models.Todo
	if err := dao.DB.First(&todo,id).Error; err != nil {
		return err
	}
	if err := dao.DB.Delete(&todo).Error; err != nil {
		return err
	}
	return nil
}