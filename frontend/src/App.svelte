<script>
  let profiles = [];
  let currentDNS = [];
  let activeIface = '';
  let platform = '';
  let loading = true;
  let selectedId = null;
  let showModal = false;
  let editMode = false;
  let editId = '';
  let formName = '';
  let formServers = [''];
  let statusMsg = '';
  let statusType = '';
  let pinging = {};
  let pingRes = {};
  let confirmDelete = null;
  let dnsActive = false;
  let dragIndex = null;
  let dragOverIndex = null;

  const APP_VERSION = '1.1.0';
  const dhcpProfile = { id: '__dhcp__', name: 'DHCP (Automatic)', servers: [] };

  $: displayProfiles = profiles ? [dhcpProfile, ...profiles] : [dhcpProfile];
  $: selected = displayProfiles.find(p => p.id === selectedId) || null;
  $: hasProfiles = profiles && profiles.length > 0;

  function updateDnsActive() {
    if (!selected) { dnsActive = false; return; }
    if (selected.id === '__dhcp__') {
      dnsActive = !currentDNS || currentDNS.length === 0;
    } else {
      dnsActive = !!(currentDNS && currentDNS.length > 0 && selected.servers.every(s => currentDNS.includes(s)));
    }
  }

  async function load() {
    loading = true;
    try {
      if (!window.go) { setStatus('Backend unavailable', 'error'); return; }
      platform = await window.go.main.App.GetPlatform();
      activeIface = await window.go.main.App.GetActiveInterface();
      currentDNS = await window.go.main.App.GetCurrentDNS();
      profiles = await window.go.main.App.GetProfiles();
      if (currentDNS && currentDNS.length > 0) {
        const matched = profiles.find(p =>
          p.servers.length === currentDNS.length &&
          p.servers.every(s => currentDNS.includes(s))
        );
        if (matched) selectedId = matched.id;
      } else {
        selectedId = '__dhcp__';
      }
      updateDnsActive();
    } catch (e) {
      setStatus('Error: ' + (e.message || e), 'error');
    } finally {
      loading = false;
    }
  }

  load();

  async function selectProfile(id) {
    selectedId = selectedId === id ? null : id;
    if (selectedId) {
      currentDNS = await window.go.main.App.GetCurrentDNS();
    }
    updateDnsActive();
  }

  async function toggleDNS() {
    if (!selected) { setStatus('Select a profile first', 'warning'); return; }
    if (selected.id === '__dhcp__') {
      setStatus('Apply a DNS profile to use custom DNS', 'warning');
      return;
    }
    if (dnsActive) {
      try {
        await window.go.main.App.RemoveDNS();
        currentDNS = [];
        setStatus('DNS removed — back to DHCP', 'success');
      } catch (e) { setStatus('Remove failed: ' + e, 'error'); }
    } else {
      try {
        await window.go.main.App.SetDNS(selected.servers);
        currentDNS = [...selected.servers];
        setStatus('Applied ' + selected.servers.join(', '), 'success');
      } catch (e) { setStatus('Apply failed: ' + e, 'error'); }
    }
    updateDnsActive();
  }

  async function applyProfile(profile) {
    selectedId = profile.id;
    if (profile.id === '__dhcp__') {
      try {
        await window.go.main.App.RemoveDNS();
        currentDNS = [];
        setStatus('DNS removed — back to DHCP', 'success');
      } catch (e) { setStatus('Remove failed: ' + e, 'error'); }
    } else {
      try {
        await window.go.main.App.SetDNS(profile.servers);
        currentDNS = [...profile.servers];
        setStatus('Applied ' + profile.servers.join(', '), 'success');
      } catch (e) { setStatus('Apply failed: ' + e, 'error'); }
    }
    updateDnsActive();
  }

  function openAdd() {
    editMode = false; editId = ''; formName = ''; formServers = [''];
    showModal = true;
  }

  function openEdit(p) {
    editMode = true; editId = p.id; formName = p.name || '';
    formServers = p.servers ? [...p.servers] : [''];
    showModal = true;
  }

  function addField() { formServers = [...formServers, '']; }

  function remField(i) { formServers = formServers.filter((_, j) => j !== i); }

  function onServerInput(i, e) {
    formServers[i] = e.target.value;
    formServers = formServers;
  }

  async function save() {
    if (!formName.trim()) { setStatus('Name is required', 'error'); return; }
    const sv = formServers.map(s => s.trim()).filter(s => s);
    if (!sv.length) { setStatus('At least one DNS server required', 'error'); return; }
    for (const s of sv) {
      if (!/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(s)) { setStatus('Invalid IP: ' + s, 'error'); return; }
    }
    if (editMode) {
      await window.go.main.App.UpdateProfile(editId, formName.trim(), sv);
    } else {
      await window.go.main.App.AddProfile(formName.trim(), sv);
    }
    showModal = false;
    profiles = await window.go.main.App.GetProfiles();
    setStatus(editMode ? 'Updated' : 'Added', 'success');
  }

  async function del(id) {
    if (confirmDelete === id) {
      await window.go.main.App.DeleteProfile(id);
      confirmDelete = null;
      if (selectedId === id) selectedId = null;
      profiles = await window.go.main.App.GetProfiles();
      setStatus('Deleted', 'success');
    } else {
      confirmDelete = id;
      setTimeout(() => { confirmDelete = null; }, 3000);
    }
  }

  function handleDragStart(e, index) {
    dragIndex = index;
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', index);
  }

  function handleDragOver(e, index) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    dragOverIndex = index;
  }

  function handleDragLeave() {
    dragOverIndex = null;
  }

  async function handleDrop(e, index) {
    e.preventDefault();
    const from = dragIndex;
    const to = index;
    if (from === null || from === to) { dragIndex = null; dragOverIndex = null; return; }
    const arr = [...profiles];
    const [moved] = arr.splice(from, 1);
    arr.splice(to, 0, moved);
    profiles = arr;
    await window.go.main.App.ReorderProfiles(profiles.map(p => p.id));
    dragIndex = null;
    dragOverIndex = null;
  }

  function handleDragEnd() {
    dragIndex = null;
    dragOverIndex = null;
  }

  async function ping(server) {
    pinging[server] = true; pinging = pinging;
    const r = await window.go.main.App.PingServer(server);
    pinging[server] = false; pinging = pinging;
    return r;
  }

  async function pingProfile(profile) {
    const servers = profile.servers || [];
    for (const s of servers) {
      const r = await ping(s);
      pingRes[s] = r; pingRes = pingRes;
    }
  }

  function setStatus(msg, type) {
    statusMsg = msg; statusType = type;
    setTimeout(() => { statusMsg = ''; statusType = ''; }, 4000);
  }
</script>

<div class="app">
  <header class="header">
    <div class="header-top">
      <div class="brand">
        <div class="logo">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="4"/><line x1="12" y1="2" x2="12" y2="6"/><line x1="12" y1="18" x2="12" y2="22"/><line x1="2" y1="12" x2="6" y2="12"/><line x1="18" y1="12" x2="22" y2="12"/>
          </svg>
        </div>
        <div>
          <h1>DNS Changer</h1>
          <span class="subtitle">{platform}{activeIface ? ' · ' + activeIface : ''}</span>
        </div>
      </div>

      <div class="dns-status">
        <span class="status-label">DNS</span>
        <div class="dns-values">
          {#if currentDNS && currentDNS.length > 0}
            {#each currentDNS as dns}
              <code>{dns}</code>
            {/each}
          {:else}
            <span class="dhcp">DHCP</span>
          {/if}
        </div>
      </div>

    </div>

    <div class="header-actions">
      <button class="btn btn-primary" on:click={openAdd}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        Add Profile
      </button>

      {#if selected}
        <div class="selected-info">
          <span class="selected-dot"></span>
          {selected.name}
        </div>
      {:else}
        <span class="select-hint">Select a profile, then toggle DNS</span>
      {/if}
    </div>
  </header>

  {#if statusMsg}
    <div class="toast toast-{statusType}">{statusMsg}</div>
  {/if}

  <main class="main">
    {#if loading}
      <div class="loading">
        <div class="spinner"></div>
        <span>Loading...</span>
      </div>
    {:else}
      <div class="profile-list">
        {#each displayProfiles as profile (profile.id)}
          {@const isDhcp = profile.id === '__dhcp__'}
          {@const isSelected = profile.id === selectedId}
          {@const profileIndex = isDhcp ? -1 : profiles.findIndex(p => p.id === profile.id)}
          {@const isActive = isDhcp ? (!currentDNS || currentDNS.length === 0) : (currentDNS && currentDNS.length > 0 && profile.servers.every(s => currentDNS.includes(s)))}
          <div
            class="profile-card"
            class:dhcp={isDhcp}
            class:selected={isSelected}
            class:dns-active={isSelected && dnsActive}
            class:dragging={!isDhcp && dragIndex === profileIndex}
            class:drag-over={!isDhcp && dragOverIndex === profileIndex && dragIndex !== profileIndex}
            draggable={!isDhcp}
            on:click={() => selectProfile(profile.id)}
            role="button"
            tabindex="0"
            on:keydown={(e) => e.key === 'Enter' && selectProfile(profile.id)}
            on:dragstart={(e) => handleDragStart(e, profileIndex)}
            on:dragover={(e) => handleDragOver(e, profileIndex)}
            on:dragleave={handleDragLeave}
            on:drop={(e) => handleDrop(e, profileIndex)}
            on:dragend={handleDragEnd}
          >
            <div class="card-left">
              {#if !isDhcp}
                <div class="drag-handle">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="8" y1="6" x2="16" y2="6"/><line x1="8" y1="12" x2="16" y2="12"/><line x1="8" y1="18" x2="16" y2="18"/></svg>
                </div>
              {/if}
              <div class="radio-circle" class:checked={isSelected}>
                {#if isSelected}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor"><path d="M20.285 2l-11.285 11.567-5.286-5.011-3.714 3.716 9 8.728 15-15.285z"/></svg>
                {/if}
              </div>
              <div class="card-info">
                <span class="card-name">{profile.name}</span>
                <div class="card-servers">
                  {#if isDhcp}
                    <span class="dhcp-label">Automatic from router / ISP</span>
                  {:else}
                    {#each profile.servers || [] as server}
                      <code>{server}</code>
                    {/each}
                  {/if}
                </div>
              </div>
            </div>

            <div class="card-right" on:click|stopPropagation>
              <button class="btn btn-apply" class:btn-primary={!isActive} class:btn-active={isActive} disabled={isActive} on:click={() => applyProfile(profile)}>
                {isActive ? 'Activated' : 'Apply'}
              </button>

              {#if !isDhcp}
                <div class="ping-results">
                  {#each profile.servers || [] as server}
                    {#if pingRes[server]}
                      <span class="ping-badge" class:ok={pingRes[server].success} class:fail={!pingRes[server].success}>
                        {pingRes[server].success ? pingRes[server].latency : '✗'}
                      </span>
                    {/if}
                  {/each}
                </div>

                <button class="icon-btn" title="Ping" on:click={() => pingProfile(profile)} disabled={Object.values(pinging).some(v => v)}>
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
                  </svg>
                </button>
                <button class="icon-btn" title="Edit" on:click={() => openEdit(profile)}>
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="icon-btn danger" title="Delete" on:click={() => del(profile.id)}>
                  {#if confirmDelete === profile.id}
                    <span class="confirm-text">Sure?</span>
                  {:else}
                    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
                  {/if}
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
      {#if !hasProfiles}
        <div class="empty">
          <p>No custom profiles yet</p>
          <button class="btn btn-primary" on:click={openAdd}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            Create your first profile
          </button>
        </div>
      {/if}
    {/if}
  </main>

  <footer class="footer">
    <span>v{APP_VERSION}</span>
  </footer>
</div>

{#if showModal}
  <div class="modal-overlay" on:click={() => showModal = false} on:keydown={(e) => e.key === 'Escape' && (showModal = false)}>
    <div class="modal" on:click|stopPropagation>
      <h2>{editMode ? 'Edit Profile' : 'New Profile'}</h2>
      <div class="form-group">
        <label for="prof-name">Name</label>
        <input id="prof-name" type="text" bind:value={formName} placeholder="e.g. Cloudflare" />
      </div>
      <div class="form-group">
        <label>DNS Servers</label>
        {#each formServers as server, i}
          <div class="server-row">
            <input type="text" value={server} on:input={(e) => onServerInput(i, e)} placeholder="e.g. 1.1.1.1" />
            {#if formServers.length > 1}
              <button class="icon-btn sm danger" on:click={() => remField(i)}>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            {/if}
          </div>
        {/each}
        <button class="btn btn-ghost sm" on:click={addField}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add Server
        </button>
      </div>
      <div class="modal-actions">
        <button class="btn btn-primary" on:click={save}>{editMode ? 'Update' : 'Save'}</button>
        <button class="btn btn-ghost" on:click={() => showModal = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .app { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

  /* ─── Header ─── */
  .header {
    padding: 20px 28px 14px;
    background: rgba(255,255,255,0.03);
    border-bottom: 1px solid var(--border);
    backdrop-filter: blur(20px);
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .header-top {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-wrap: wrap;
  }

  .brand { display: flex; align-items: center; gap: 12px; }
  .logo {
    width: 38px; height: 38px;
    border-radius: 10px;
    background: var(--primary-dim);
    color: var(--primary);
    display: flex; align-items: center; justify-content: center;
  }
  .brand h1 { font-size: 17px; font-weight: 700; letter-spacing: -0.3px; line-height: 1.2; }
  .subtitle { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.6px; }

  .dns-status {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-left: auto;
  }
  .status-label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; }
  .dns-values { display: flex; gap: 4px; }
  .dns-values code {
    padding: 2px 8px; border-radius: 5px;
    background: var(--surface); border: 1px solid var(--border);
    font-size: 12px; font-family: monospace;
  }
  .dhcp { font-size: 12px; color: var(--text-muted); font-style: italic; padding: 2px 0; }

  /* ─── Header actions ─── */
  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .selected-info {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 500;
    color: var(--primary);
  }
  .selected-dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--primary);
    animation: pulse-dot 1.8s infinite;
  }
  @keyframes pulse-dot { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }
  .select-hint { font-size: 12px; color: var(--text-muted); }

  /* ─── Buttons ─── */
  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 7px 14px;
    border-radius: var(--radius-sm);
    border: none;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s;
    letter-spacing: 0.2px;
  }
  .btn-primary { background: var(--primary); color: #000; }
  .btn-primary:hover { filter: brightness(1.15); }
  .btn-ghost { background: transparent; color: var(--text); border: 1px solid var(--border); }
  .btn-ghost:hover { background: var(--surface-hover); border-color: rgba(255,255,255,0.15); }
  .btn.sm { padding: 5px 10px; font-size: 11px; }

  .icon-btn {
    width: 30px; height: 30px;
    border-radius: 7px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: all 0.15s;
  }
  .icon-btn:hover { background: var(--surface-hover); color: var(--text); }
  .icon-btn.danger:hover { background: rgba(251,113,133,0.12); color: var(--error); }
  .icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
  .icon-btn.sm { width: 24px; height: 24px; }
  .confirm-text { font-size: 10px; font-weight: 700; color: var(--error); }
  .btn-apply { padding: 8px 18px; font-size: 12px; font-weight: 700; min-width: 72px; justify-content: center; letter-spacing: 0.3px; }
  .btn-active { background: var(--success); color: #000; }
  .btn-active:hover { filter: brightness(1.15); }
  .btn-active:disabled { opacity: 0.6; cursor: default; filter: none; }

  /* ─── Toast ─── */
  .toast {
    position: fixed; top: 16px; right: 16px;
    padding: 10px 18px; border-radius: var(--radius-sm);
    font-size: 13px; font-weight: 500;
    z-index: 100;
    animation: slideIn 0.2s ease;
    box-shadow: 0 8px 30px rgba(0,0,0,0.4);
    backdrop-filter: blur(10px);
  }
  .toast-success { background: rgba(74,222,128,0.15); color: var(--success); border: 1px solid rgba(74,222,128,0.2); }
  .toast-error { background: rgba(251,113,133,0.15); color: var(--error); border: 1px solid rgba(251,113,133,0.2); }
  .toast-warning { background: rgba(251,191,36,0.15); color: var(--warning); border: 1px solid rgba(251,191,36,0.2); }
  @keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }

  /* ─── Footer ─── */
  .footer {
    padding: 12px 28px 20px;
    text-align: center;
    font-size: 11px;
    color: var(--text-muted);
    border-top: 1px solid transparent;
  }

  /* ─── Main ─── */
  .main { flex: 1; overflow-y: auto; min-height: 0; padding: 20px 28px; max-width: 720px; width: 100%; margin: 0 auto; }

  /* ─── Loading ─── */
  .loading {
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 14px; padding: 80px 0;
    color: var(--text-muted); font-size: 13px;
  }
  .spinner {
    width: 24px; height: 24px; border-radius: 50%;
    border: 2px solid rgba(255,255,255,0.06);
    border-top-color: var(--primary);
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ─── Empty ─── */
  .empty {
    display: flex; flex-direction: column;
    align-items: center; gap: 12px;
    padding: 80px 0;
    color: var(--text-muted);
  }
  .empty p { font-size: 15px; }

  /* ─── Profile List ─── */
  .profile-list { display: flex; flex-direction: column; gap: 8px; }

  .profile-card {
    display: flex; align-items: center; justify-content: space-between;
    padding: 14px 16px;
    border-radius: var(--radius);
    background: var(--surface);
    border: 1px solid var(--border);
    cursor: pointer;
    transition: all 0.15s;
    user-select: none;
  }
  .profile-card:hover { background: var(--surface-hover); }
  .profile-card.selected {
    border-color: var(--primary);
    background: var(--primary-dim);
    box-shadow: 0 0 0 1px rgba(56,189,248,0.3), 0 4px 20px rgba(56,189,248,0.08);
  }
  .profile-card.dns-active {
    border-color: var(--success);
    box-shadow: 0 0 0 1px rgba(74,222,128,0.25), 0 4px 20px rgba(74,222,128,0.06);
  }
  .profile-card.dhcp {
    border-style: dashed;
    border-color: rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.02);
  }
  .profile-card.dhcp:hover { background: rgba(255,255,255,0.05); }
  .profile-card.dhcp.selected {
    border-style: dashed;
    border-color: var(--primary);
    background: var(--primary-dim);
    box-shadow: 0 0 0 1px rgba(56,189,248,0.3), 0 4px 20px rgba(56,189,248,0.08);
  }
  .profile-card.dhcp.dns-active {
    border-style: dashed;
    border-color: var(--success);
    box-shadow: 0 0 0 1px rgba(74,222,128,0.25), 0 4px 20px rgba(74,222,128,0.06);
  }
  .dhcp-label { font-size: 11px; color: var(--text-muted); font-style: italic; }

  .card-left { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }

  .radio-circle {
    width: 20px; height: 20px; border-radius: 50%;
    border: 2px solid rgba(255,255,255,0.15);
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; transition: all 0.15s;
  }
  .radio-circle.checked {
    border-color: var(--primary);
    background: var(--primary);
    color: #000;
  }
  .profile-card.dns-active .radio-circle.checked { border-color: var(--success); background: var(--success); }

  .card-info { min-width: 0; }
  .card-name { font-size: 14px; font-weight: 600; display: block; margin-bottom: 3px; }
  .card-servers { display: flex; gap: 4px; flex-wrap: wrap; }
  .card-servers code {
    padding: 1px 7px; border-radius: 4px;
    background: rgba(255,255,255,0.05);
    font-size: 11px; font-family: monospace;
    color: var(--text-muted);
  }
  .profile-card.selected .card-servers code { background: rgba(56,189,248,0.1); }

  .card-right { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }

  .profile-card.dragging { opacity: 0.4; }
  .profile-card.drag-over { border-top: 2px solid var(--primary); }
  .drag-handle {
    display: flex; align-items: center; cursor: grab;
    color: var(--text-muted); padding: 4px; margin-right: 4px;
    border-radius: 4px; flex-shrink: 0;
  }
  .drag-handle:hover { background: var(--surface-hover); color: var(--text); }
  .profile-card:active .drag-handle { cursor: grabbing; }

  .ping-results { display: flex; gap: 3px; }
  .ping-badge {
    padding: 1px 6px; border-radius: 4px;
    font-size: 10px; font-weight: 700;
  }
  .ping-badge.ok { color: var(--success); background: rgba(74,222,128,0.1); }
  .ping-badge.fail { color: var(--error); background: rgba(251,113,133,0.1); }

  /* ─── Modal ─── */
  .modal-overlay {
    position: fixed; inset: 0;
    background: rgba(0,0,0,0.7);
    display: flex; align-items: center; justify-content: center;
    z-index: 50;
    backdrop-filter: blur(4px);
  }
  .modal {
    background: #14171f;
    border-radius: var(--radius);
    padding: 24px;
    width: 440px; max-width: 90vw;
    border: 1px solid var(--border);
    box-shadow: 0 20px 60px rgba(0,0,0,0.6);
  }
  .modal h2 { font-size: 16px; font-weight: 700; margin-bottom: 20px; }

  .form-group { margin-bottom: 16px; }
  .form-group label {
    display: block; font-size: 11px; color: var(--text-muted);
    margin-bottom: 6px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  input[type="text"] {
    width: 100%; padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: rgba(255,255,255,0.04);
    color: var(--text); font-size: 13px;
    outline: none; transition: border-color 0.15s;
  }
  input[type="text"]:focus { border-color: var(--primary); }

  .server-row { display: flex; gap: 8px; margin-bottom: 8px; }
  .server-row input { flex: 1; }

  .modal-actions {
    display: flex; gap: 8px;
    justify-content: flex-end; margin-top: 20px;
  }
</style>
