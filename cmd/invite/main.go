package main

const (
	Unanswered = iota
	Yes
	No
)

type Invite struct {
	ID       int
	User     int
	Response int
}

var users = map[int]map[Invite]struct{}{
	0: make(map[Invite]struct{}),
	1: make(map[Invite]struct{}),
	2: make(map[Invite]struct{}),
	3: make(map[Invite]struct{}),
	4: make(map[Invite]struct{}),
	5: make(map[Invite]struct{}),
	6: make(map[Invite]struct{}),
	7: make(map[Invite]struct{}),
}

func sendinvite(iv Invite) {
	users[iv.User][iv] = struct{}{}
}

func main() {

}
