package model

type AnimationResult struct {
	Result     []*Animation `json:"results"`
	Pagination *Pagination  `json:"pagination"`
}
type Pagination struct {
	Limit      int `json:"limit"`
	Page       int `json:"page"`
	NumPages   int `json:"num_pages"`
	NumResults int `json:"num_results"`
}

type Motion struct {
	MotionId  string `json:"motion_id"`
	ProductId string `json:"product_id"`
	Name      string `json:"name"`
}

type Animation struct {
	Id                string    `json:"id"`
	Type              string    `json:"type"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	CharacterType     string    `json:"character_type"`
	Name              string    `json:"name"`
	Thumbnail         string    `json:"thumbnail"`
	ThumbnailAnimated string    `json:"thumbnail_animated"`
	MotionId          string    `json:"motion_id"`
	Motions           []*Motion `json:"motions"`
	Source            string    `json:"source"`
}

