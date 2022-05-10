package model

type Author struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique"`
}

type Book struct {
	ID       int       `json:"id" gorm:"primaryKey"`
	Title    string    `json:"title" gorm:"unique"`
	AuthorID int       `json:"-"`
	Author   *Author   `json:"author"`
	Reviews  []*Review `json:"reviews"`
}

type NewAuthor struct {
	Name string `json:"name"`
}

type NewBook struct {
	Title      string `json:"title"`
	AuthorName string `json:"authorName"`
}

type Review struct {
	ID     int    `json:"id" gorm:"primaryKey"`
	BookID int    `json:"-"`
	Star   int    `json:"star"`
	Text   string `json:"text"`
}
