package model

type Author struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique"`
}

type AuthorFilter struct {
	ID   *int    `json:"id"`
	Name *string `json:"name"`
}

type Book struct {
	ID       int       `json:"id" gorm:"primaryKey"`
	Title    string    `json:"title" gorm:"unique"`
	AuthorID int       `json:"-"`
	Author   *Author   `json:"author"`
	Reviews  []*Review `json:"reviews"`
}

type BookFilter struct {
	ID         *int    `json:"id"`
	Title      *string `json:"title"`
	AuthorName *string `json:"authorName"`
	MinStar    *int    `json:"minStar"`
	MaxStar    *int    `json:"maxStar"`
}

type NewAuthor struct {
	Name string `json:"name"`
}

type NewBook struct {
	Title      string `json:"title"`
	AuthorName string `json:"authorName"`
}

type NewReview struct {
	BookID int    `json:"bookId"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
}

type Review struct {
	ID     int    `json:"id" gorm:"primaryKey"`
	BookID int    `json:"-"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
}

type ReviewFilter struct {
	MinStar *int `json:"minStar"`
	MaxStar *int `json:"maxStar"`
}
