package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	// "os"
	tea "github.com/charmbracelet/bubbletea"
	"unsafe"
)

type Proc struct {
	Pid         int
	Exe         string
	Description string
	Path        string
	Parent      int
}

type model struct {
	procs  []Proc
	cursor int
	height int
	min    int
	max    int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height - 6
		m.max = m.height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.min {
					m.min--
					m.max--
				}
			}
		case "down", "j":
			if m.cursor < len(m.procs) {
				m.cursor++
				if m.cursor > m.max {
					m.min++
					m.max++
				}
			}
			// case "enter", " ":
			// 	_, ok := m.selected[m.cursor]
			// 	if ok {
			// 		delete(m.selected, m.cursor)
			// 	} else {
			// 		m.selected[m.cursor] = struct{}{}
			// }
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Processes\n\n"

	for i, proc := range m.procs {
		if i < m.min {
			continue
		}
		if i > m.max {
			break
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		name := proc.Exe
		if proc.Description != "" {
			name = proc.Description + " (" + name + ")"
		}

		s += fmt.Sprintf("%s [%6d] %s\n", cursor, proc.Pid, name)
	}

	s += "\nPress q to quit.\n"

	return s
}

var procEntry = windows.ProcessEntry32{}
var modEntry = windows.ModuleEntry32{}

func closeHandle(h windows.Handle, hType string) {
	err := windows.CloseHandle(h)
	if err != nil {
		fmt.Printf("Failed to close %s handle: %v", hType, err)
		os.Exit(1)
	}
}

func AddModuleInfo(p *Proc) bool {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE, uint32(p.Pid))
	if err != nil {
		return false
	}
	defer closeHandle(handle, "module snapshot")

	err = windows.Module32First(handle, &modEntry)
	if err != nil {
		return false
	}

	p.Path = windows.UTF16ToString(modEntry.ExePath[:])
	return true
}

func AddVersionInfo(p *Proc) {
	var zp windows.Handle = 0
	size, err := windows.GetFileVersionInfoSize(p.Path, &zp)
	if err != nil {
		return
	}
	buf := make([]byte, size)
	bufPtr := unsafe.Pointer(&buf[0])
	err = windows.GetFileVersionInfo(p.Path, 0, size, bufPtr)
	if err != nil {
		return
	}

	// NOTE(tad): only supporting default translation
	var bufOffset uintptr
	var valLen uint32
	err = windows.VerQueryValue(
		bufPtr,
		`\VarFileInfo\Translation`,
		unsafe.Pointer(&bufOffset),
		&valLen)
	if err != nil || valLen < 4 {
		return
	}
	start := int(bufOffset) - int(uintptr(bufPtr))
	t := buf[start:(start + 4)]
	t[0], t[1] = t[1], t[0]
	t[2], t[3] = t[3], t[2]
	translation := fmt.Sprintf("%x", t)

	err = windows.VerQueryValue(
		bufPtr,
		`\StringFileInfo\`+translation+`\`+"FileDescription",
		unsafe.Pointer(&bufOffset),
		&valLen)
	if err != nil {
		return
	}
	start = int(bufOffset) - int(uintptr(bufPtr))
	d := buf[start:]
	utf16 := make([]uint16, valLen)
	for i := range utf16 {
		idx := i * 2
		utf16[i] = uint16(d[idx+1])<<8 | uint16(d[idx])
	}
	p.Description = windows.UTF16ToString(utf16)
}

func main() {
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	modEntry.Size = uint32(unsafe.Sizeof(modEntry))

	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		fmt.Println("Failed to get process snapshot: ", err)
		os.Exit(1)
	}
	defer closeHandle(handle, "process snapshot")

	procs := make([]Proc, 0, 256)
	err = windows.Process32First(handle, &procEntry)
	if err != nil {
		fmt.Println("Failed to get process entries: ", err)
		os.Exit(1)
	}

	for err == nil {
		if procEntry.ProcessID != 0 {
			proc := Proc{
				Pid:    int(procEntry.ProcessID),
				Exe:    windows.UTF16ToString(procEntry.ExeFile[:]),
				Parent: int(procEntry.ParentProcessID),
			}
			if AddModuleInfo(&proc) {
				AddVersionInfo(&proc)
			}
			procs = append(procs, proc)
		}

		err = windows.Process32Next(handle, &procEntry)
	}

	// TODO(tad): use channels + WMI to watch for new processes and killed processes

	p := tea.NewProgram(model{procs: procs})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
