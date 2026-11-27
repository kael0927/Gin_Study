package main
import (
	"fmt"
	"strings"
)
type Reader interface {
	Read() string
}
type Writer interface {
	Writer(s string) error
}
type Buffer struct {
	data strings.Builder
}
type ReadWriter interface {
	Reader
	Writer
}
func (b *Buffer) Read() string {
	return b.data.String()
}
func (b *Buffer) Writer(s string) error {
	b.data.WriteString(s)
	return nil
}

func PrintAll(r Reader) {
	fmt.Println("读取内容：", r.Read())
}
func Copy(dst ReadWriter, src Reader) error {
	return dst.Writer(src.Read())
}
func main() {
	var buf Buffer
	_ = buf.Writer("hele")
	var rw ReadWriter = &buf
	PrintAll(rw)
	Copy(rw, rw)
	PrintAll(rw)
	var _ ReadWriter = (*Buffer)(nil)
}