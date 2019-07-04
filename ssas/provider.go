package ssas

import "log"

func Provide() string {
	log.Print("I will provide")
	return "stuff"
}
