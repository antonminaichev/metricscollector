package memstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateCounter(t *testing.T) {
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name    string
		storage *MemStorage
		args    args
		want    map[string]int64
	}{
		{
			name: "Add new metric",
			storage: &MemStorage{
				Counter: map[string]int64{},
			},
			args: args{
				name:  "test",
				value: 1,
			},
			want: map[string]int64{
				"test": 1,
			},
		},
		{
			name: "Increase existing metric",
			storage: &MemStorage{
				Counter: map[string]int64{
					"test": 1,
				},
			},
			args: args{
				name:  "test",
				value: 2,
			},
			want: map[string]int64{
				"test": 3,
			},
		},
		{
			name: "Add negative number",
			storage: &MemStorage{
				Counter: map[string]int64{
					"test": 1,
				},
			},
			args: args{
				name:  "test",
				value: -2,
			},
			want: map[string]int64{
				"test": -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.storage.UpdateCounter(tt.args.name, tt.args.value)
			require.Equal(t, tt.storage.Counter, tt.want)
		})
	}
}

func TestUpdateGauge(t *testing.T) {
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name    string
		storage *MemStorage
		args    args
		want    map[string]float64
	}{
		{
			name: "Add new metric",
			storage: &MemStorage{
				Gauge: map[string]float64{},
			},
			args: args{
				name:  "test",
				value: 1.535,
			},
			want: map[string]float64{
				"test": 1.535,
			},
		},
		{
			name: "Change existing metric",
			storage: &MemStorage{
				Gauge: map[string]float64{
					"test": 1.51,
				},
			},
			args: args{
				name:  "test",
				value: 2.003,
			},
			want: map[string]float64{
				"test": 2.003,
			},
		},
		{
			name: "Add negative number",
			storage: &MemStorage{
				Gauge: map[string]float64{
					"test": 1.51,
				},
			},
			args: args{
				name:  "test",
				value: -2.583,
			},
			want: map[string]float64{
				"test": -2.583,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.storage.UpdateGauge(tt.args.name, tt.args.value)
			require.Equal(t, tt.storage.Gauge, tt.want)
		})
	}
}

func TestMemStorageGetCounter(t *testing.T) {
	tests := []struct {
		name    string
		storage *MemStorage
		want    map[string]int64
	}{
		{
			name: "Get counter",
			storage: &MemStorage{
				Counter: map[string]int64{
					"test":  1,
					"test2": 2,
				},
			},
			want: map[string]int64{"test": 1, "test2": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.storage.GetCounter(), tt.want)
		})
	}
}

func TestMemStorageGetGauge(t *testing.T) {
	tests := []struct {
		name    string
		storage *MemStorage
		want    map[string]float64
	}{
		{
			name: "Get counter",
			storage: &MemStorage{
				Gauge: map[string]float64{
					"test":  1.352,
					"test2": 2.432,
				},
			},
			want: map[string]float64{"test": 1.352, "test2": 2.432},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.storage.GetGauge(), tt.want)
		})
	}
}
