package conf

import "testing"

// TestTraceSampleRatioZeroDisablesSampling 验证 sample_ratio=0 不再被强制改为 1
// （OPEN_ITEMS C8）：0 表示不采样，负值/大于 1 才是非法值。
func TestTraceSampleRatioZeroDisablesSampling(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"关闭采样", 0, 0},
		{"部分采样", 0.1, 0.1},
		{"负值回落全量", -1, 1},
		{"大于 1 回落全量", 2, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{}
			c.Trace.SampleRatio = tc.in
			applyDefaults(c)
			if c.Trace.SampleRatio != tc.want {
				t.Errorf("SampleRatio = %v, want %v", c.Trace.SampleRatio, tc.want)
			}
		})
	}
}
