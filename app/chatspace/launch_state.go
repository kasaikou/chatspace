package chatspace

type status int

const (
	statusClosed status = iota
	statusWork
	statusChat
)

type launchState struct {
	status        status
	workMemberIDs []string
	chatCha
}
