package config_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	libConfig "github.com/MiG-21/go-lib-config"
)

var _ = Describe("Config", func() {
	Context("EnvReader", func() {
		It("Type test should be Ok", func() {
			defer os.Clearenv()

			vars := map[string]string{
				"TEST_INTEGER":         "-5",
				"TEST_UNSINTEGER":      "5",
				"TEST_FLOAT":           "5.5",
				"TEST_BOOLEAN":         "true",
				"TEST_STRING":          "test",
				"TEST_DURATION":        "1h5m10s",
				"TEST_TIME":            "2012-04-23T18:25:43.511Z",
				"TEST_ARRAYINT":        "1,2,3",
				"TEST_ARRAYSTRING":     "a,b,c",
				"TEST_MAPSTRINGINT":    "a:1,b:2,c:3",
				"TEST_MAPSTRINGSTRING": "a:x,b:y,c:z",
			}
			setEnv(vars)

			type TestAllTypesCfg struct {
				Integer         int64             `env:"TEST_INTEGER"`
				UnsInteger      uint64            `env:"TEST_UNSINTEGER"`
				Float           float64           `env:"TEST_FLOAT"`
				Boolean         bool              `env:"TEST_BOOLEAN"`
				String          string            `env:"TEST_STRING"`
				Duration        time.Duration     `env:"TEST_DURATION"`
				Time            time.Time         `env:"TEST_TIME"`
				ArrayInt        []int             `env:"TEST_ARRAYINT"`
				ArrayString     []string          `env:"TEST_ARRAYSTRING"`
				MapStringInt    map[string]int    `env:"TEST_MAPSTRINGINT"`
				MapStringString map[string]string `env:"TEST_MAPSTRINGSTRING"`
			}

			var cfg TestAllTypesCfg
			reader := libConfig.NewEnvReader()
			err := reader.Read(&cfg)
			Expect(err).NotTo(HaveOccurred())

			expected := TestAllTypesCfg{
				Integer:     -5,
				UnsInteger:  5,
				Float:       5.5,
				Boolean:     true,
				String:      "test",
				Duration:    durationFunc("1h5m10s"),
				Time:        timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
				ArrayInt:    []int{1, 2, 3},
				ArrayString: []string{"a", "b", "c"},
				MapStringInt: map[string]int{
					"a": 1,
					"b": 2,
					"c": 3,
				},
				MapStringString: map[string]string{
					"a": "x",
					"b": "y",
					"c": "z",
				},
			}

			Expect(cfg).To(Equal(expected))
		})

		It("Time test should be Ok", func() {
			defer os.Clearenv()

			vars := map[string]string{
				"TEST_TIME1": "2021-02-25T11:11:11.511Z",
				"TEST_TIME2": "Thu Feb 25 11:11:11 2021",
				"TEST_TIME3": "Jan 1 11:11:11",
				"TEST_TIME4": "2021-01-25T11:11:11.511Z|2021-02-25T11:11:11.511Z",
				"TEST_TIME5": "a:2021-01-25T11:11:11.511Z|b:2021-02-25T11:11:11.511Z",
			}
			setEnv(vars)

			type TestTimeCfg struct {
				Time1 time.Time            `env:"TEST_TIME1"`
				Time2 time.Time            `env:"TEST_TIME2" data-layout:"Mon Jan _2 15:04:05 2006"`
				Time3 time.Time            `env:"TEST_TIME3" data-layout:"Jan _2 15:04:05"`
				Time4 []time.Time          `env:"TEST_TIME4" data-separator:"|"`
				Time5 map[string]time.Time `env:"TEST_TIME5" data-separator:"|"`
			}

			var cfg TestTimeCfg
			reader := libConfig.NewEnvReader()
			err := reader.Read(&cfg)
			Expect(err).NotTo(HaveOccurred())

			expected := TestTimeCfg{
				Time1: timeFunc("2021-02-25T11:11:11.511Z", time.RFC3339),
				Time2: timeFunc("Thu Feb 25 11:11:11 2021", time.ANSIC),
				Time3: timeFunc("Jan 1 11:11:11", time.Stamp),
				Time4: []time.Time{
					timeFunc("2021-01-25T11:11:11.511Z", time.RFC3339),
					timeFunc("2021-02-25T11:11:11.511Z", time.RFC3339),
				},
				Time5: map[string]time.Time{
					"a": timeFunc("2021-01-25T11:11:11.511Z", time.RFC3339),
					"b": timeFunc("2021-02-25T11:11:11.511Z", time.RFC3339),
				},
			}

			Expect(cfg).To(Equal(expected))
		})
	})

	Context("NewConfigService", func() {
		It("Read and refresh", func() {
			defer os.Clearenv()

			vars := map[string]string{
				"TEST_VAR1": "1",
				"TEST_VAR2": "2",
				"TEST_VAR3": "3",
			}
			setEnv(vars)

			type TestCfg struct {
				Var1 int `env:"TEST_VAR1"`
				Var2 int `env:"TEST_VAR2"`
				Var3 int `env:"TEST_VAR3"`
			}
			var cfg TestCfg
			service := libConfig.NewConfigService(5 * time.Millisecond)
			reader := libConfig.NewEnvReader()
			err := service.Start(&cfg, nil, &reader)
			defer func() {
				_ = service.Stop()
			}()
			Expect(err).NotTo(HaveOccurred())

			expected := TestCfg{
				Var1: 1,
				Var2: 2,
				Var3: 3,
			}
			Expect(cfg).To(Equal(expected))

			vars = map[string]string{
				"TEST_VAR1": "3",
				"TEST_VAR2": "2",
				"TEST_VAR3": "1",
			}
			setEnv(vars)
			time.Sleep(7 * time.Millisecond)
			expected = TestCfg{
				Var1: 3,
				Var2: 2,
				Var3: 1,
			}
			Expect(cfg).To(Equal(expected))
		})
	})
})
