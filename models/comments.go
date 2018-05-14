package models

type Comment struct {
	Id   string
	Text string
	Date string
}

func NewComment(id, text, date string) *Comment {
	return &Comment{id, text, date}
}
