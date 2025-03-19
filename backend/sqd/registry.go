package sqd

const EvmRegistry = "https://cdn.subsquid.io/archives/solana.json"

type ArchiveProvider struct {
	Provider      string `json:"provider"`
	DataSourceUrl string `json:"dataSourceUrl"`
	Release       string `json:"release"`

	// Incomplete
}

type ArchiveEntry struct {
	Id        string            `json:"id"`
	ChainName string            `json:"chainName"`
	IsTestnet bool              `json:"isTestnet"`
	Network   string            `json:"network"`
	Providers []ArchiveProvider `json:"providers"`
}

type ArchiveRegistryResponse struct {
	Archives []ArchiveEntry `json:"archives"`
}
