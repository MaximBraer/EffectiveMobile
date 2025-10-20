package postgres

type SQLDataBase struct {
	Server          string `yaml:"server" json:"server"`
	Database        string `yaml:"database" json:"database"`
	MaxIdleCons     int    `yaml:"max_idle_cons" json:"max_idle_cons"`
	MaxOpenCons     int    `yaml:"max_open_cons" json:"max_open_cons"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	Port            string `yaml:"port" json:"port"`
}
