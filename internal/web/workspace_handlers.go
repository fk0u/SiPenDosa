package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sipen/internal/store"

	"github.com/go-chi/chi/v5"
)

// WorkspacesPageHandler displays the workspaces management and switching interface
func (h *Handlers) WorkspacesPageHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	isSuper := isPrivileged(user)
	workspaces, err := h.store.ListWorkspacesForUser(user.ID, isSuper)
	if err != nil {
		workspaces = []store.WorkspaceDetail{}
	}

	activeWs, _ := h.store.GetActiveWorkspace(user.ID)
	var activeMembers []store.WorkspaceMemberInfo
	if activeWs != nil {
		activeMembers, _ = h.store.ListWorkspaceMembers(activeWs.ID)
	}

	pd := PageData{
		Title:      "Ruang Kerja (Workspaces) — SiPenDosa Server",
		ActivePage: "workspaces",
		User:       user,
		Data: map[string]interface{}{
			"Workspaces":      workspaces,
			"ActiveWorkspace": activeWs,
			"ActiveMembers":   activeMembers,
			"IsSuperadmin":    isSuper,
		},
	}
	h.populateWorkspaceContext(r, &pd)
	h.renderer.Render(w, r, "workspaces/index.html", pd)
}

// SwitchWorkspaceHandler switches the active workspace for the authenticated user
func (h *Handlers) SwitchWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		idStr = r.FormValue("workspace_id")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
		return
	}

	// Verify user has access to this workspace
	isSuper := isPrivileged(user)
	if !isSuper {
		role, err := h.store.GetWorkspaceMemberRole(id, user.ID)
		if err != nil || role == "" {
			http.Error(w, "Akses ditolak: Anda bukan anggota workspace ini", http.StatusForbidden)
			return
		}
	}

	_ = h.store.SetActiveWorkspace(user.ID, id)
	h.store.AddActivityLog("system", "Beralih Workspace", fmt.Sprintf("User: %s -> Workspace ID: %d", user.Username, id))

	referer := r.Header.Get("Referer")
	if referer != "" && !strings.Contains(referer, "/workspaces") {
		http.Redirect(w, r, referer, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// CreateWorkspaceHandler creates a new collaborative workspace
func (h *Handlers) CreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	description := strings.TrimSpace(r.FormValue("description"))
	gatewayMode := r.FormValue("gateway_mode")
	if gatewayMode != "workspace_shared" {
		gatewayMode = "personal"
	}

	if name == "" {
		http.Redirect(w, r, "/workspaces?error=Nama+workspace+wajib+diisi", http.StatusSeeOther)
		return
	}

	ws := &store.Workspace{
		Name:        name,
		Slug:        slug,
		Description: description,
		OwnerID:     user.ID,
		GatewayMode: gatewayMode,
	}

	// If workspace_shared is chosen, default shared gateway to creator's WhatsApp session
	if gatewayMode == "workspace_shared" {
		ws.SharedWAUserID = sql.NullInt64{Int64: user.ID, Valid: true}
	}

	created, err := h.store.CreateWorkspace(ws)
	if err != nil {
		http.Redirect(w, r, "/workspaces?error="+err.Error(), http.StatusSeeOther)
		return
	}

	h.store.AddActivityLog("system", "Membuat Workspace Baru", fmt.Sprintf("Workspace: %s (%s) oleh %s", created.Name, created.Slug, user.Username))
	http.Redirect(w, r, "/workspaces?success=Workspace+berhasil+dibuat", http.StatusSeeOther)
}

// UpdateWorkspaceHandler updates workspace settings (gateway mode, description, etc.)
func (h *Handlers) UpdateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
		return
	}

	ws, err := h.store.GetWorkspaceByID(id)
	if err != nil || ws == nil {
		http.Redirect(w, r, "/workspaces?error=Workspace+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	// Verify permission: only owner or superadmin can update
	if ws.OwnerID != user.ID && !isPrivileged(user) {
		http.Error(w, "Forbidden: Hanya pemilik workspace atau admin yang dapat mengubah pengaturan", http.StatusForbidden)
		return
	}

	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name != "" {
		ws.Name = name
	}
	ws.Description = strings.TrimSpace(r.FormValue("description"))

	gatewayMode := r.FormValue("gateway_mode")
	if gatewayMode == "shared" || gatewayMode == "workspace_shared" {
		ws.GatewayMode = "shared"
		sharedUID, _ := strconv.ParseInt(r.FormValue("shared_wa_user_id"), 10, 64)
		if sharedUID <= 0 {
			sharedUID = user.ID
		}
		ws.SharedWAUserID = sql.NullInt64{Int64: sharedUID, Valid: true}
	} else {
		ws.GatewayMode = "personal"
		ws.SharedWAUserID = sql.NullInt64{Valid: false}
	}
	ws.SharedPhone = strings.TrimSpace(r.FormValue("shared_phone"))

	if err := h.store.UpdateWorkspace(ws); err != nil {
		http.Redirect(w, r, "/workspaces?error="+err.Error(), http.StatusSeeOther)
		return
	}

	h.store.AddActivityLog("system", "Pengaturan Workspace Diperbarui", fmt.Sprintf("Workspace: %s, Mode Gateway: %s", ws.Name, ws.GatewayMode))
	http.Redirect(w, r, "/workspaces?success=Pengaturan+workspace+berhasil+disimpan", http.StatusSeeOther)
}

// DeleteWorkspaceHandler removes a workspace
func (h *Handlers) DeleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
		return
	}

	ws, err := h.store.GetWorkspaceByID(id)
	if err != nil || ws == nil {
		http.Redirect(w, r, "/workspaces?error=Workspace+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	if ws.OwnerID != user.ID && !isPrivileged(user) {
		http.Error(w, "Forbidden: Hanya pemilik workspace yang dapat menghapus", http.StatusForbidden)
		return
	}

	if err := h.store.DeleteWorkspace(id); err != nil {
		http.Redirect(w, r, "/workspaces?error="+err.Error(), http.StatusSeeOther)
		return
	}

	// Switch active workspace to another valid one
	_ = h.store.SetActiveWorkspace(user.ID, 1)

	h.store.AddActivityLog("system", "Menghapus Workspace", fmt.Sprintf("Workspace ID %d dihapus oleh %s", id, user.Username))
	http.Redirect(w, r, "/workspaces?success=Workspace+berhasil+dihapus", http.StatusSeeOther)
}

// AddWorkspaceMemberHandler adds a user into the workspace
func (h *Handlers) AddWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	targetUsername := strings.TrimSpace(r.FormValue("username"))
	role := r.FormValue("role")
	if role != "admin" && role != "viewer" {
		role = "member"
	}

	targetUser, err := h.store.GetUserByUsername(targetUsername)
	if err != nil || targetUser == nil {
		http.Redirect(w, r, fmt.Sprintf("/workspaces?error=Pengguna+'%s'+tidak+ditemukan", targetUsername), http.StatusSeeOther)
		return
	}

	if err := h.store.AddWorkspaceMember(id, targetUser.ID, role); err != nil {
		http.Redirect(w, r, "/workspaces?error="+err.Error(), http.StatusSeeOther)
		return
	}

	h.store.AddActivityLog("system", "Menambahkan Anggota Workspace", fmt.Sprintf("User: %s ditambahkan ke Workspace ID %d sebagai %s", targetUsername, id, role))
	http.Redirect(w, r, "/workspaces?success=Anggota+berhasil+ditambahkan", http.StatusSeeOther)
}

// RemoveWorkspaceMemberHandler removes a user from the workspace
func (h *Handlers) RemoveWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	wsID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	targetUserID, _ := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if wsID <= 0 || targetUserID <= 0 {
		http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
		return
	}

	ws, err := h.store.GetWorkspaceByID(wsID)
	if err != nil || ws == nil {
		http.Redirect(w, r, "/workspaces?error=Workspace+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	// Cannot remove the owner of the workspace
	if targetUserID == ws.OwnerID {
		http.Redirect(w, r, "/workspaces?error=Pemilik+workspace+tidak+dapat+dikeluarkan", http.StatusSeeOther)
		return
	}

	// Must be owner, admin, or superadmin, or removing oneself
	callerRole, _ := h.store.GetWorkspaceMemberRole(wsID, user.ID)
	if ws.OwnerID != user.ID && callerRole != "admin" && !isPrivileged(user) && user.ID != targetUserID {
		http.Error(w, "Forbidden: Anda tidak memiliki wewenang untuk mengeluarkan anggota ini", http.StatusForbidden)
		return
	}

	if err := h.store.RemoveWorkspaceMember(wsID, targetUserID); err != nil {
		http.Redirect(w, r, "/workspaces?error="+err.Error(), http.StatusSeeOther)
		return
	}

	h.store.AddActivityLog("system", "Mengeluarkan Anggota Workspace", fmt.Sprintf("User ID %d dikeluarkan dari Workspace ID %d oleh %s", targetUserID, wsID, user.Username))
	http.Redirect(w, r, "/workspaces?success=Anggota+berhasil+dikeluarkan", http.StatusSeeOther)
}
