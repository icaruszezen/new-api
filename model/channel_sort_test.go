package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pages over channels that share a sort value must not overlap or skip any: ties are ordered by id, newest first.
func TestChannelListPagesOrderTiedChannelsByID(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
	t.Cleanup(func() { DB.Exec("DELETE FROM channels") })

	priorities := []int64{0, 5, 0, 5, 0, 0, 5}
	ids := make([]int, len(priorities))
	for i := range priorities {
		channel := Channel{
			Name:     fmt.Sprintf("tied-%d", i),
			Key:      "sk-test",
			Status:   common.ChannelStatusEnabled,
			Priority: &priorities[i],
		}
		require.NoError(t, DB.Create(&channel).Error)
		ids[i] = channel.Id
	}

	tests := []struct {
		name    string
		options []ChannelSortOptions
		want    []int
	}{
		{
			name: "default priority descending",
			want: []int{ids[6], ids[3], ids[1], ids[5], ids[4], ids[2], ids[0]},
		},
		{
			name:    "priority ascending",
			options: []ChannelSortOptions{NewChannelSortOptions("priority", "asc", false)},
			want:    []int{ids[5], ids[4], ids[2], ids[0], ids[6], ids[3], ids[1]},
		},
		{
			name:    "balance, equal for every channel",
			options: []ChannelSortOptions{NewChannelSortOptions("balance", "desc", false)},
			want:    []int{ids[6], ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const pageSize = 2
			var listed []int
			for start := 0; start < len(ids)+pageSize; start += pageSize {
				page, err := GetAllChannels(start, pageSize, false, false, tt.options...)
				require.NoError(t, err)
				for _, channel := range page {
					listed = append(listed, channel.Id)
				}
			}
			assert.Equal(t, tt.want, listed)
		})
	}
}
