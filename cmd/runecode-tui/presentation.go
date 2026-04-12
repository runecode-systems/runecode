package main

type contentPresentationMode string

const (
	presentationRendered   contentPresentationMode = "rendered"
	presentationRaw        contentPresentationMode = "raw"
	presentationStructured contentPresentationMode = "structured"
)

func nextPresentationMode(mode contentPresentationMode) contentPresentationMode {
	switch mode {
	case presentationRendered:
		return presentationRaw
	case presentationRaw:
		return presentationStructured
	default:
		return presentationRendered
	}
}

func normalizePresentationMode(mode contentPresentationMode) contentPresentationMode {
	switch mode {
	case presentationRendered, presentationRaw, presentationStructured:
		return mode
	default:
		return presentationRendered
	}
}
