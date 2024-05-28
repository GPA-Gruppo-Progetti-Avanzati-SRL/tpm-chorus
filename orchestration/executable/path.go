package executable

import "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"

type Path struct {
	Cfg config.Path
}

func NewPath(cfg config.Path) (Path, error) {
	return Path{Cfg: cfg}, nil
}
