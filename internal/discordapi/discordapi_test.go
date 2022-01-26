package discordapi

import (
	"reflect"
	"testing"

	"github.com/alexisgeoffrey/aoe4elobot/pkg/aoeapi"
	"github.com/bwmarrin/discordgo"
)

func TestMessageCreate(t *testing.T) {
	type args struct {
		s *discordgo.Session
		m *discordgo.MessageCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MessageCreate(tt.args.s, tt.args.m)
		})
	}
}

func TestUpdateAllElo(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		guildId string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateAllElo(tt.args.s, tt.args.guildId)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAllElo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UpdateAllElo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMemberElo(t *testing.T) {
	type args struct {
		st      *discordgo.State
		u       user
		guildId string
	}
	tests := []struct {
		name    string
		args    args
		want    aoeapi.UserElo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMemberElo(tt.args.st, tt.args.u, tt.args.guildId)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMemberElo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMemberElo() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func Test_removeAllEloRoles(t *testing.T) {
// 	type args struct {
// 		s       *discordgo.Session
// 		guildId string
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := removeAllEloRoles(tt.args.s, tt.args.guildId); (err != nil) != tt.wantErr {
// 				t.Errorf("removeAllEloRoles() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }
