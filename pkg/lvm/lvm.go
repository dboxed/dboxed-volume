package lvm

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/dboxed/dboxed-volume/pkg/util"
)

type PVEntry struct {
	PvName string `json:"pv_name"`
	VgName string `json:"vg_name"`
	PvFmt  string `json:"pv_fmt"`
	PvAttr string `json:"pv_attr"`
	PvSize string `json:"pv_size"`
	PvFree string `json:"pv_free"`
}

type pvsReport struct {
	Report []struct {
		Pv []PVEntry `json:"pv"`
	} `json:"report"`
}

type VGEntry struct {
	VgName    string `json:"vg_name"`
	PvCount   string `json:"pv_count"`
	LvCount   string `json:"lv_count"`
	SnapCount string `json:"snap_count"`
	VgAttr    string `json:"vg_attr"`
	VgSize    string `json:"vg_size"`
	VgFree    string `json:"vg_free"`
}
type vgsReport struct {
	Report []struct {
		Vg []VGEntry `json:"vg"`
	} `json:"report"`
}

type LVEntry struct {
	LvName          string `json:"lv_name"`
	VgName          string `json:"vg_name"`
	LvAttr          string `json:"lv_attr"`
	LvSize          string `json:"lv_size"`
	PoolLv          string `json:"pool_lv"`
	Origin          string `json:"origin"`
	DataPercent     string `json:"data_percent"`
	MetadataPercent string `json:"metadata_percent"`
	MovePv          string `json:"move_pv"`
	MirrorLog       string `json:"mirror_log"`
	CopyPercent     string `json:"copy_percent"`
	ConvertLv       string `json:"convert_lv"`

	LvTags string `json:"lv_tags"`
}

type lvsReport struct {
	Report []struct {
		Lv []LVEntry `json:"lv"`
	} `json:"report"`
}

func buildColNames[T any]() []string {
	var ret []string
	t := reflect.TypeFor[T]()
	for i := range t.NumField() {
		f := t.Field(i)
		jt := f.Tag.Get("json")
		if jt != "" {
			ret = append(ret, jt)
		}
	}
	return ret
}

func ListPVs() ([]PVEntry, error) {
	h, err := util.RunCommandJson[pvsReport]("pvs", "--reportformat=json", "-o", strings.Join(buildColNames[PVEntry](), ","))
	if err != nil {
		return nil, err
	}
	return h.Report[0].Pv, nil
}

func ListVGs() ([]VGEntry, error) {
	h, err := util.RunCommandJson[vgsReport]("vgs", "--reportformat=json", "-o", strings.Join(buildColNames[VGEntry](), ","))
	if err != nil {
		return nil, err
	}
	return h.Report[0].Vg, nil
}

func ListLVs() ([]LVEntry, error) {
	h, err := util.RunCommandJson[lvsReport]("lvs", "--reportformat=json", "-o", strings.Join(buildColNames[LVEntry](), ","))
	if err != nil {
		return nil, err
	}
	return h.Report[0].Lv, nil
}

func PVCreate(dev string) error {
	err := util.RunCommand("pvcreate", dev)
	if err != nil {
		return err
	}
	return nil
}

func VGCreate(vgName string, devs ...string) error {
	args := []string{
		vgName,
	}
	args = append(args, devs...)
	err := util.RunCommand("vgcreate", args...)
	if err != nil {
		return err
	}
	return nil
}

func VGGet(vgName string) (*VGEntry, error) {
	vgs, err := ListVGs()
	if err != nil {
		return nil, err
	}
	for _, vg := range vgs {
		if vg.VgName == vgName {
			return &vg, nil
		}
	}
	return nil, os.ErrNotExist
}

func VGDeactivate(vgName string) error {
	err := util.RunCommand("vgchange", "-an", vgName)
	if err != nil {
		return err
	}
	return nil
}

func TPCreate100(vgName string, tpName string, tags []string) error {
	args := []string{
		"-l100%FREE",
		"--thinpool", tpName,
		vgName,
	}
	for _, t := range tags {
		args = append(args, "--addtag", t)
	}
	err := util.RunCommand("lvcreate", args...)
	if err != nil {
		return err
	}
	return nil
}

func LVGet(vgName string, lvName string) (*LVEntry, error) {
	lvs, err := ListLVs()
	if err != nil {
		return nil, err
	}
	for _, lv := range lvs {
		if lv.VgName == vgName && lv.LvName == lvName {
			return &lv, nil
		}
	}
	return nil, os.ErrNotExist
}

func LVCreate(vgName string, lvName string, size int64, tags []string) error {
	args := []string{
		"--name", lvName,
		"-L", fmt.Sprintf("%dB", size),
		vgName,
	}
	for _, t := range tags {
		args = append(args, "--addtag", t)
	}
	err := util.RunCommand("lvcreate", args...)
	if err != nil {
		return err
	}
	return nil
}

func TLVCreate(vgName string, tpName string, lvName string, size int64, tags []string) error {
	args := []string{
		"--name", lvName,
		"--thin",
		"-V", fmt.Sprintf("%dB", size),
		fmt.Sprintf("%s/%s", vgName, tpName),
	}
	for _, t := range tags {
		args = append(args, "--addtag", t)
	}
	err := util.RunCommand("lvcreate", args...)
	if err != nil {
		return err
	}
	return nil
}

func TLVSnapCreate(vgName string, lvName string, tpName string, snapName string) error {
	args := []string{
		"--name", snapName,
		"--type", "thin",
		"--thinpool", tpName,
		fmt.Sprintf("%s/%s", vgName, lvName),
	}
	err := util.RunCommand("lvcreate", args...)
	if err != nil {
		return err
	}
	return nil
}

func LVRemove(vgName string, lvName string) error {
	err := util.RunCommand("lvremove", fmt.Sprintf("%s/%s", vgName, lvName), "-f")
	if err != nil {
		return err
	}
	return nil
}

func LVActivate(vgName string, lvName string, activate bool) error {
	args := []string{
		"-K",
	}
	if activate {
		args = append(args, "-ay")
	} else {
		args = append(args, "-an")
	}
	args = append(args, fmt.Sprintf("%s/%s", vgName, lvName))
	err := util.RunCommand("lvchange", args...)
	if err != nil {
		return err
	}
	return nil
}

func FindPVLVs(pvName string) ([]LVEntry, error) {
	pvs, err := ListPVs()
	if err != nil {
		return nil, err
	}

	var foundPv *PVEntry
	for _, pv := range pvs {
		if pv.PvName == pvName {
			foundPv = &pv
			break
		}
	}
	if foundPv == nil {
		return nil, fmt.Errorf("physical volume %s not found", pvName)
	}
	if foundPv.VgName == "" {
		return nil, fmt.Errorf("physical volume %s seems to not have a volume group", pvName)
	}

	vgs, err := ListVGs()
	if err != nil {
		return nil, err
	}

	var foundVg *VGEntry
	for _, vg := range vgs {
		if vg.VgName == foundPv.VgName {
			foundVg = &vg
			break
		}
	}
	if foundVg == nil {
		return nil, fmt.Errorf("volume group %s not found in list of volume groups", foundPv.VgName)
	}

	lvs, err := ListLVs()
	if err != nil {
		return nil, err
	}

	var ret []LVEntry
	for _, lv := range lvs {
		if lv.VgName == foundVg.VgName {
			ret = append(ret, lv)
		}
	}
	return ret, nil
}
