package php

type phpServerContextKey string

const (
	environmentContextKey    phpServerContextKey = "env"
	responseWriterContextKey phpServerContextKey = "rw"
)
