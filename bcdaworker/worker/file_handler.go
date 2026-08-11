package worker

import "os"

func CreateDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0744); err != nil {
			return err
		}
		return err
	} else if err != nil {
		return err
	}
	return nil
}
