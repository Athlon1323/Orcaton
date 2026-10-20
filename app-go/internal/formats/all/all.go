package all

import (
	"orcaton/internal/formats"
	"orcaton/internal/formats/drt"
	"orcaton/internal/formats/xmeml"
)

func init() {
	formats.RegisterImporter(xmeml.Importer{})
	formats.RegisterImporter(drt.Importer{})
	formats.RegisterExporter(xmeml.Exporter{})
	formats.RegisterExporter(drt.DefaultExporter)
}
