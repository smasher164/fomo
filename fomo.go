package fomo

import "time"

type User struct {
	ID    int
	First string
	Last  string
}

type FacebookUser struct {
	User
	FacebookID int
}

type Event struct {
	ID          int
	Description string
	Datetime    time.Time
}

type Status int

const (
	Unanswered Status = iota
	Accepted
	Rejected
	Invited
)

func AddFacebookUser() {

}
