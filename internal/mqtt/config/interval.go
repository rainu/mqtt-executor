package config

import (
	"strings"
	"time"
)

type Interval time.Duration

func (i *Interval) UnmarshalJSON(b []byte) error {
	duration, err := time.ParseDuration(strings.Replace(string(b), `"`, "", -1))
	if err != nil {
		return err
	}

	*i = Interval(duration)
	return nil
}
