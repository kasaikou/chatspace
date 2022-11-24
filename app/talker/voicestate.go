package talker

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type GuildVCMonitor struct {
	lock            sync.Mutex
	guildID         string
	ignoreMemberIDs map[string]struct{}
	joinMembers     map[string]map[string]*discordgo.VoiceState
	memberStates    map[string]*discordgo.VoiceState
}

type VoiceStateUpdateType int

const (
	VoiceStateUpdateTypeOther VoiceStateUpdateType = iota
	VoiceStateUpdateTypeJoin
	VoiceStateUpdateTypeLeave
)

func NewGuildVCMonitor(guildID string, ignoreMemberIDs []string) *GuildVCMonitor {

	monitor := GuildVCMonitor{
		guildID:         guildID,
		ignoreMemberIDs: map[string]struct{}{},
		joinMembers:     map[string]map[string]*discordgo.VoiceState{},
		memberStates:    map[string]*discordgo.VoiceState{},
	}

	for _, ignoreMemberID := range ignoreMemberIDs {
		monitor.ignoreMemberIDs[ignoreMemberID] = struct{}{}
	}

	return &monitor
}

func (gvm *GuildVCMonitor) VoiceStateUpdate(vsu discordgo.VoiceStateUpdate) VoiceStateUpdateType {
	gvm.lock.Lock()
	defer gvm.lock.Unlock()

	if vsu.GuildID != gvm.guildID {
		return VoiceStateUpdateTypeOther
	} else if _, exist := gvm.ignoreMemberIDs[vsu.UserID]; exist {
		return VoiceStateUpdateTypeOther
	}

	gvm.memberStates[vsu.UserID] = vsu.VoiceState

	if vsu.BeforeUpdate != nil {
		switch {
		case vsu.BeforeUpdate.ChannelID == "":
			break

		case vsu.BeforeUpdate.ChannelID == vsu.ChannelID:
			prevMembers := gvm.joinMembers[vsu.ChannelID]
			prevMembers[vsu.UserID] = vsu.VoiceState
			gvm.joinMembers[vsu.ChannelID] = prevMembers
			return VoiceStateUpdateTypeOther

		default:
			prevMembers := gvm.joinMembers[vsu.BeforeUpdate.ChannelID]
			delete(prevMembers, vsu.UserID)
			gvm.joinMembers[vsu.BeforeUpdate.ChannelID] = prevMembers
		}
	}

	prevMembers := gvm.joinMembers[vsu.ChannelID]
	if prevMembers == nil {
		prevMembers = map[string]*discordgo.VoiceState{}
	}
	if vsu.ChannelID != "" {
		prevMembers[vsu.UserID] = vsu.VoiceState
		gvm.joinMembers[vsu.ChannelID] = prevMembers
		return VoiceStateUpdateTypeJoin

	} else {
		delete(prevMembers, vsu.UserID)
		gvm.joinMembers[vsu.ChannelID] = prevMembers
		return VoiceStateUpdateTypeLeave

	}
}

func (gvm *GuildVCMonitor) NumJoinedMembers() int {
	gvm.lock.Lock()
	defer gvm.lock.Unlock()

	count := 0
	for _, channels := range gvm.joinMembers {
		count += len(channels)
	}

	return count
}

func (gvm *GuildVCMonitor) VCJoinedMemberIDs(channelID string) []*discordgo.VoiceState {
	gvm.lock.Lock()
	defer gvm.lock.Unlock()

	results := []*discordgo.VoiceState{}
	if members, exist := gvm.joinMembers[channelID]; !exist {
		for _, member := range members {
			results = append(results, member)
		}
		return results
	}

	return results
}

func (gvm *GuildVCMonitor) MemberVoiceState(userID string) *discordgo.VoiceState {
	gvm.lock.Lock()
	defer gvm.lock.Unlock()
	return gvm.memberStates[userID]
}
