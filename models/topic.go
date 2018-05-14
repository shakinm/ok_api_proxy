package models

type Topic struct {
	Id             string
	Title          string
	Image          string
	Like_count     float64
	Comments_count float64
	Comments       map[string]*Comment
}

func NewTopic(id, title, image string, like_count, comments_count float64, comments map[string]*Comment) *Topic {
	return &Topic{id, title, image, like_count, comments_count, comments}
}
