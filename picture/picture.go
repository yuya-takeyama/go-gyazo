package picture

import "time"

type Picture struct {
	Id        int
	Hash      string
	UserId    string
	Body      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}
