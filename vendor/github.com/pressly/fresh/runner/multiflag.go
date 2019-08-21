package runner

import "fmt"

type Multiflag []string

func (m *Multiflag) String() string {
	return fmt.Sprint(*m)
}

func (m *Multiflag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
