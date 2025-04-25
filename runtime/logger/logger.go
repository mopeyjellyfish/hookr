package logger

type Logger func(msg string)

func Default(msg string) {
	println(msg)
}
