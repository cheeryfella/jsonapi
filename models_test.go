package jsonapi_test

import (
	"fmt"
	"time"

	"github.com/cheeryfella/jsonapi"
)

type BadModel struct {
	ID int `jsonapi:"primary"`
}

type ModelBadTypes struct {
	ID           string     `jsonapi:"primary,badtypes"`
	StringField  string     `jsonapi:"attr,string_field"`
	FloatField   float64    `jsonapi:"attr,float_field"`
	TimeField    time.Time  `jsonapi:"attr,time_field"`
	TimePtrField *time.Time `jsonapi:"attr,time_ptr_field"`
}

type WithPointer struct {
	ID       *uint64  `jsonapi:"primary,with-pointers"`
	Name     *string  `jsonapi:"attr,name"`
	IsActive *bool    `jsonapi:"attr,is-active"`
	IntVal   *int     `jsonapi:"attr,int-val"`
	FloatVal *float32 `jsonapi:"attr,float-val"`
}

type Numeric struct {
	ID    string    `jsonapi:"primary,numeric"`
	Int   int       `jsonapi:"attr,int,omitempty"`
	Uint  uint      `jsonapi:"attr,uint,omitempty"`
	Float float64   `jsonapi:"attr,float,omitempty"`
	Cmplx complex64 `jsonapi:"attr,cmplx,omitempty"`
}

type Timestamp struct {
	ID   int        `jsonapi:"primary,timestamps"`
	Time time.Time  `jsonapi:"attr,timestamp,iso8601"`
	Next *time.Time `jsonapi:"attr,next,iso8601"`
}

type Car struct {
	ID    *string `jsonapi:"primary,cars"`
	Make  *string `jsonapi:"attr,make,omitempty"`
	Model *string `jsonapi:"attr,model,omitempty"`
	Year  *uint   `jsonapi:"attr,year,omitempty"`
}

type Post struct {
	Blog
	ID            uint64     `jsonapi:"primary,posts"`
	BlogID        int        `jsonapi:"attr,blog_id"`
	ClientID      string     `jsonapi:"client-id"`
	Title         string     `jsonapi:"attr,title"`
	Body          string     `jsonapi:"attr,body"`
	Comments      []*Comment `jsonapi:"relation,comments"`
	LatestComment *Comment   `jsonapi:"relation,latest_comment"`
}

type Comment struct {
	ID       int    `jsonapi:"primary,comments"`
	ClientID string `jsonapi:"client-id"`
	PostID   int    `jsonapi:"attr,post_id"`
	Body     string `jsonapi:"attr,body"`
}

type Book struct {
	ID          uint64  `jsonapi:"primary,books"`
	Author      string  `jsonapi:"attr,author"`
	ISBN        string  `jsonapi:"attr,isbn"`
	Title       string  `jsonapi:"attr,title,omitempty"`
	Description *string `jsonapi:"attr,description"`
	Pages       *uint   `jsonapi:"attr,pages,omitempty"`
	PublishedAt time.Time
	Tags        []string `jsonapi:"attr,tags"`
}

type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	ClientID      string    `jsonapi:"client-id"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

func (b *Blog) JSONAPILinks() *jsonapi.Links {
	return &jsonapi.Links{
		"self": fmt.Sprintf("https://example.com/api/blogs/%d", b.ID),
		"comments": jsonapi.Link{
			Href: fmt.Sprintf("https://example.com/api/blogs/%d/comments", b.ID),
			Meta: jsonapi.Meta{
				"counts": map[string]uint{
					"likes":    4,
					"comments": 20,
				},
			},
		},
	}
}

func (b *Blog) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
	if relation == "posts" {
		return &jsonapi.Links{
			"related": jsonapi.Link{
				Href: fmt.Sprintf("https://example.com/api/blogs/%d/posts", b.ID),
				Meta: jsonapi.Meta{
					"count": len(b.Posts),
				},
			},
		}
	}
	if relation == "current_post" {
		return &jsonapi.Links{
			"self": fmt.Sprintf("https://example.com/api/posts/%s", "3"),
			"related": jsonapi.Link{
				Href: fmt.Sprintf("https://example.com/api/blogs/%d/current_post", b.ID),
			},
		}
	}
	return nil
}

func (b *Blog) JSONAPIMeta() *jsonapi.Meta {
	return &jsonapi.Meta{
		"detail": "extra details regarding the blog",
	}
}

func (b *Blog) JSONAPIRelationshipMeta(relation string) *jsonapi.Meta {
	if relation == "posts" {
		return &jsonapi.Meta{
			"this": map[string]interface{}{
				"can": map[string]interface{}{
					"go": []interface{}{
						"as",
						"deep",
						map[string]interface{}{
							"as": "required",
						},
					},
				},
			},
		}
	}
	if relation == "current_post" {
		return &jsonapi.Meta{
			"detail": "extra current_post detail",
		}
	}
	return nil
}

type BadComment struct {
	ID   uint64 `jsonapi:"primary,bad-comment"`
	Body string `jsonapi:"attr,body"`
}

func (bc *BadComment) JSONAPILinks() *jsonapi.Links {
	return &jsonapi.Links{
		"self": []string{"invalid", "should error"},
	}
}

type Company struct {
	ID        string    `jsonapi:"primary,companies"`
	Name      string    `jsonapi:"attr,name"`
	Boss      Employee  `jsonapi:"attr,boss"`
	Teams     []Team    `jsonapi:"attr,teams"`
	FoundedAt time.Time `jsonapi:"attr,founded-at,iso8601"`
}

type Team struct {
	Name    string     `jsonapi:"attr,name"`
	Leader  *Employee  `jsonapi:"attr,leader"`
	Members []Employee `jsonapi:"attr,members"`
}

type Employee struct {
	Firstname string     `jsonapi:"attr,firstname"`
	Surname   string     `jsonapi:"attr,surname"`
	Age       int        `jsonapi:"attr,age"`
	HiredAt   *time.Time `jsonapi:"attr,hired-at,iso8601"`
}

type CustomIntType int
type CustomFloatType float64
type CustomStringType string

type CustomAttributeTypes struct {
	ID string `jsonapi:"primary,customtypes"`

	Int        CustomIntType  `jsonapi:"attr,int"`
	IntPtr     *CustomIntType `jsonapi:"attr,intptr"`
	IntPtrNull *CustomIntType `jsonapi:"attr,intptrnull"`

	Float  CustomFloatType  `jsonapi:"attr,float"`
	String CustomStringType `jsonapi:"attr,string"`
}
