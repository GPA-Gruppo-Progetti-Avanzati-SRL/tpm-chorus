package executable

import "tpm-chorus/orchestration/config"

type Path struct {
	Cfg config.Path
}

func NewPath(cfg config.Path) (Path, error) {
	return Path{Cfg: cfg}, nil
}
