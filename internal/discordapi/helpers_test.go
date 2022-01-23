package discordapi

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func Test_formatUpdateMessage(t *testing.T) {
	type args struct {
		st      *discordgo.State
		u       []user
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
			got, err := formatUpdateMessage(tt.args.st, tt.args.u, tt.args.guildId)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatUpdateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("formatUpdateMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveToConfig(t *testing.T) {
	type args struct {
		m *discordgo.MessageCreate
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
			got, err := saveToConfig(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("saveToConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("saveToConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
