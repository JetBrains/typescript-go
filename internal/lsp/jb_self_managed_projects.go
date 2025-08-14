package lsp

import (
	"sync"
	"time"

	"github.com/microsoft/typescript-go/internal/project"
)

type SelfManagedProjectInfo struct {
	Project          *project.Project
	LastAccessed     int64
	FilesLastUpdated map[string]int64
}

var (
	mu                sync.Mutex
	projectInfoByName = make(map[string]*SelfManagedProjectInfo)
)

func GetAllSelfManagedProjects() []*project.Project {
	cleanupStaleProjects()

	mu.Lock()
	defer mu.Unlock()

	var projects []*project.Project
	for _, info := range projectInfoByName {
		projects = append(projects, info.Project)
	}
	return projects
}

func GetSelfManagedProjectForFile(s *Server, projectFileName string, file string) *project.Project {
	defer cleanupStaleProjects()
	return getProjectForFileImpl(s, projectFileName, file)
}

func getProjectForFileImpl(s *Server, projectFileName string, file string) *project.Project {
	nowMs := time.Now().UnixMilli()
	fileMod := getFileModTimeMs(s, file)

	mu.Lock()
	defer mu.Unlock()

	if info := projectInfoByName[projectFileName]; info != nil {
		prevMod, hadPrev := info.FilesLastUpdated[file]
		if !hadPrev || fileMod == prevMod {
			info.LastAccessed = nowMs
			return info.Project
		}

		info.Project.Close()
		delete(projectInfoByName, projectFileName)
	}

	newProject := createNewSelfManagedProject(s, projectFileName, file)
	if newProject == nil {
		return nil
	}

	newInfo := &SelfManagedProjectInfo{
		Project:          newProject,
		LastAccessed:     nowMs,
		FilesLastUpdated: make(map[string]int64),
	}
	newInfo.FilesLastUpdated[file] = fileMod
	projectInfoByName[projectFileName] = newInfo

	return newProject
}

func cleanupStaleProjects() {
	nowMs := time.Now().UnixMilli()
	const ttl = int64(5 * time.Minute / time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	for name, info := range projectInfoByName {
		if nowMs-info.LastAccessed > ttl {
			info.Project.Close()
			delete(projectInfoByName, name)
		}
	}
}

func getFileModTimeMs(s *Server, file string) int64 {
	if !s.FS().FileExists(file) {
		return 0
	}
	fi := s.FS().Stat(file)
	if fi == nil {
		return 0
	}
	return fi.ModTime().UnixMilli()
}

func createNewSelfManagedProject(s *Server, projectFileName string, file string) *project.Project {
	p := project.NewConfiguredProject(projectFileName, s.projectService.ToPath(projectFileName), s.projectService)
	if p.GetProgram().GetSourceFile(file) == nil {
		s.Log("Project: ", projectFileName, " doesn't contain the file: ", file)
		return nil
	}

	if err := p.LoadConfig(); err != nil {
		s.Log("Could not load configured project:", projectFileName, err)
		return nil
	}

	return p
}
