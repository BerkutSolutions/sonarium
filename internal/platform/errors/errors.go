package errors

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e AppError) Error() string {
	return e.Message
}

func NewBadRequest(message string) AppError {
	return AppError{
		Code:       "invalid_request",
		Message:    message,
		HTTPStatus: 400,
	}
}

func NewNotFound(message string) AppError {
	return AppError{
		Code:       "not_found",
		Message:    message,
		HTTPStatus: 404,
	}
}

func NewInternal(message string) AppError {
	return AppError{
		Code:       "internal_error",
		Message:    message,
		HTTPStatus: 500,
	}
}
