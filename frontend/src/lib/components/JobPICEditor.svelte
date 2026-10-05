<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import { api, type JobPIC } from '$lib/api';

 let { jobId, jobName = '', pics, onchange }: { jobId: number; jobName?: string; pics: JobPIC[]; onchange: (pics: JobPIC[]) => void } = $props();

 let open = $state(false);
 let search = $state('');
 let users = $state<JobPIC[]>([]);
 let busy = $state(false);
 let loading = $state(false);
 let error = $state('');
 let success = $state('');
 let dialogEl = $state<HTMLDialogElement | null>(null);
 let searchInput = $state<HTMLInputElement | null>(null);

 // ponytail: derive available users by excluding currently assigned PICs
 let options = $derived(users.filter(user => !pics.some(pic => pic.userId === user.userId)));

 // Debounce user search API when modal dialog is open
 $effect(() => {
  const query = search;
  if (!open) return;
  let cancelled = false;
  loading = true;
  const timer = setTimeout(async () => {
   try {
    const result = await api<{ users: JobPIC[] }>(`/jobs/pic-users?q=${encodeURIComponent(query)}`);
    if (!cancelled) {
     users = result.users;
     error = '';
    }
   } catch (e) {
    if (!cancelled) error = e instanceof Error ? e.message : 'Akun tidak dapat dimuat.';
   } finally {
    if (!cancelled) loading = false;
   }
  }, 200);
  return () => {
   cancelled = true;
   clearTimeout(timer);
  };
 });

 // ponytail: standard HTML5 dialog with declarative effect for showModal / close
 $effect(() => {
  if (open) {
   dialogEl?.showModal();
   search = '';
   setTimeout(() => searchInput?.focus(), 60);
  } else {
   dialogEl?.close();
  }
 });

 async function change(userId: string, remove = false) {
  if (busy) return;
  busy = true;
  error = '';
  success = '';
  try {
   const result = await api<{ pics: JobPIC[] }>(
    `/jobs/${jobId}/pics${remove ? `/${userId}` : ''}`,
    {
     method: remove ? 'DELETE' : 'POST',
     ...(remove ? {} : { body: JSON.stringify({ userId }) })
    }
   );
   onchange(result.pics);
   success = remove ? 'PIC dihapus.' : 'PIC ditambahkan.';
   setTimeout(() => { success = ''; }, 3000);
  } catch (e) {
   error = e instanceof Error ? e.message : 'PIC tidak dapat disimpan.';
  } finally {
   busy = false;
  }
 }

 function openModal() {
  error = '';
  success = '';
  open = true;
 }

 function closeModal() {
  if (busy) return;
  open = false;
 }

 function handleBackdropClick(event: MouseEvent) {
  if (event.target === dialogEl && !busy) {
   closeModal();
  }
 }
</script>

<!-- Inline PIC representation in table cell -->
<div class="pic-section">
 <div class="pic-list">
  <span class="pic-tag-label" title={$translate("Person in Charge")}>
   <svg class="pic-icon" viewBox="0 0 20 20" fill="currentColor" width="13" height="13" aria-hidden="true">
    <path d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z"/>
   </svg>
   {$translate("PIC:")}
  </span>
  {#each pics as pic (pic.userId)}
   <span class="pic-chip" class:inactive={!pic.isActive} title={pic.name + (pic.isActive ? '' : ' (' + $translate('Nonaktif') + ')')}>
    <span class="avatar-circle">{pic.name.trim().charAt(0).toUpperCase()}</span>
    <span class="chip-label">{pic.name}</span>
    <button
     type="button"
     class="chip-remove"
     disabled={busy}
     aria-label={`Hapus PIC ${pic.name}`}
     onclick={(e) => { e.stopPropagation(); void change(pic.userId, true); }}
    >×</button>
   </span>
  {:else}
   <span class="no-pic-label">{$translate("Belum ada")}</span>
  {/each}
  <button
   type="button"
   class="add-pic-trigger"
   aria-haspopup="dialog"
   aria-expanded={open}
   onclick={openModal}
  >
   <span class="plus-icon">+</span>
   <span>{$translate("Tambah PIC")}</span>
  </button>
 </div>
</div>

<!-- Modal Pop-up with smooth fade-in animation -->
<dialog
 bind:this={dialogEl}
 class="pic-dialog"
 oncancel={(e) => { if (busy) e.preventDefault(); else open = false; }}
 onclick={handleBackdropClick}
 aria-labelledby={`pic-title-${jobId}`}
>
 <div class="dialog-content">
  <header class="dialog-header">
   <div class="dialog-title-area">
    <div class="dialog-badge">
     <svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14" aria-hidden="true">
      <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z"/>
     </svg>
     {$translate("Manajemen PIC")}
    </div>
    <h2 id={`pic-title-${jobId}`} class="dialog-title">{$translate("Kelola Person In Charge")}</h2>
    {#if jobName}
     <p class="dialog-job-name">{jobName}</p>
    {/if}
   </div>
   <button
    type="button"
    class="dialog-close-btn"
    disabled={busy}
    onclick={closeModal}
    aria-label={$translate("Tutup")}
   >×</button>
  </header>

  {#if error}
   <div class="dialog-alert error" role="alert">
    <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
     <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
    </svg>
    <span>{$translate(error)}</span>
   </div>
  {/if}

  {#if success}
   <div class="dialog-alert success" role="status">
    <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
     <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
    </svg>
    <span>{$translate(success)}</span>
   </div>
  {/if}

  <!-- Current Assigned PIC Section -->
  <div class="dialog-section">
   <div class="section-header">
    <span class="section-title">{$translate("PIC Terdaftar")}</span>
    <span class="count-badge">{pics.length}</span>
   </div>
   <div class="assigned-pic-list">
    {#each pics as pic (pic.userId)}
     <div class="assigned-card" class:inactive={!pic.isActive}>
      <div class="assigned-info">
       <span class="avatar-large">{pic.name.trim().charAt(0).toUpperCase()}</span>
       <div class="assigned-names">
        <span class="assigned-fullname">{pic.name}</span>
        <span class="assigned-meta">
         @{pic.username}
         {#if !pic.isActive}
          <span class="inactive-tag">{$translate("Nonaktif")}</span>
         {/if}
        </span>
       </div>
      </div>
      <button
       type="button"
       class="btn-remove-pic"
       disabled={busy}
       onclick={() => void change(pic.userId, true)}
       title={$translate("Hapus PIC")}
      >
       <svg viewBox="0 0 20 20" fill="currentColor" width="14" height="14" aria-hidden="true">
        <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd"/>
       </svg>
       <span>{$translate("Hapus")}</span>
      </button>
     </div>
    {:else}
     <p class="empty-state">{$translate("Belum ada PIC yang ditugaskan pada pekerjaan ini.")}</p>
    {/each}
   </div>
  </div>

  <!-- Add New PIC Section -->
  <div class="dialog-section">
   <div class="section-header">
    <label for={`search-pic-${jobId}`} class="section-title">{$translate("Tambah PIC Baru")}</label>
   </div>
   <div class="search-box">
    <svg class="search-icon" viewBox="0 0 20 20" fill="currentColor" width="16" height="16" aria-hidden="true">
     <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd"/>
    </svg>
    <input
     id={`search-pic-${jobId}`}
     bind:this={searchInput}
     type="search"
     bind:value={search}
     placeholder={$translate("Nama, username, atau email")}
     disabled={busy}
    />
    {#if search}
     <button type="button" class="btn-clear-search" onclick={() => search = ''} aria-label="Bersihkan pencarian">×</button>
    {/if}
   </div>

   <div class="candidates-area">
    {#if loading}
     <div class="loading-state" role="status">
      <span class="spinner"></span>
      <span>{$translate("Memuat akun…")}</span>
     </div>
    {:else if options.length}
     <div class="candidates-list">
      {#each options as user (user.userId)}
       <div class="candidate-row">
        <div class="candidate-info">
         <span class="avatar-large candidate-avatar">{user.name.trim().charAt(0).toUpperCase()}</span>
         <div class="candidate-names">
          <span class="candidate-name">{user.name}</span>
          <span class="candidate-username">@{user.username}</span>
         </div>
        </div>
        <button
         type="button"
         class="btn-add-candidate"
         disabled={busy}
         onclick={() => void change(user.userId)}
        >
         <span class="plus-icon">+</span>
         <span>{$translate("Pilih")}</span>
        </button>
       </div>
      {/each}
     </div>
    {:else}
     <p class="empty-state">{$translate("Tidak ada akun lain yang cocok.")}</p>
    {/if}
    <small class="search-hint">{$translate("Ketik nama untuk mencari akun lainnya.")}</small>
   </div>
  </div>

  <footer class="dialog-footer">
   <button type="button" class="btn-done" disabled={busy} onclick={closeModal}>
    {$translate("Selesai")}
   </button>
  </footer>
 </div>
</dialog>

<style>
 /* ponytail: zero-dep inline PIC chips and modal dialog with native fade-in */
 .pic-section {
  margin-top: 6px;
 }
 .pic-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
 }
 .pic-tag-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--his-muted, #595959);
  text-transform: uppercase;
  letter-spacing: 0.04em;
 }
 .pic-icon {
  color: var(--his-primary, #007a41);
 }
 .no-pic-label {
  font-size: 0.78rem;
  color: #8c8c8c;
  font-style: italic;
 }
 .pic-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--his-primary-soft, #e8f8ef);
  border: 1px solid #c2ebd4;
  border-radius: 999px;
  padding: 2px 7px 2px 4px;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--his-primary-dark, #005c31);
  max-width: 100%;
  transition: all 140ms ease;
 }
 .pic-chip.inactive {
  background: #f3f4f6;
  border-color: #e5e7eb;
  color: #6b7280;
 }
 .avatar-circle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--his-primary, #007a41);
  color: #ffffff;
  font-size: 0.65rem;
  font-weight: 700;
  line-height: 1;
  flex-shrink: 0;
 }
 .pic-chip.inactive .avatar-circle {
  background: #9ca3af;
 }
 .chip-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 130px;
 }
 .chip-remove {
  /* Override global button sizing */
  min-width: unset !important;
  min-height: unset !important;
  width: 16px !important;
  height: 16px !important;
  padding: 0 !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 50%;
  background: rgba(0, 92, 49, 0.12);
  color: var(--his-primary-dark, #005c31);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  cursor: pointer;
  transition: background-color 140ms ease, color 140ms ease;
 }
 .chip-remove:hover:not(:disabled) {
  background: var(--his-danger, #c52327);
  color: #ffffff;
 }
 .add-pic-trigger {
  /* Override global button sizing */
  min-width: unset !important;
  min-height: unset !important;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px !important;
  border-radius: 999px;
  border: 1px dashed var(--his-primary, #007a41);
  background: #ffffff;
  color: var(--his-primary, #007a41);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 160ms ease;
 }
 .add-pic-trigger:hover {
  background: var(--his-primary-soft, #e8f8ef);
  border-style: solid;
  transform: translateY(-1px);
 }
 .plus-icon {
  font-size: 0.85rem;
  font-weight: 800;
  line-height: 1;
 }

 /* Modal Pop-up with smooth fade-in animations */
 .pic-dialog {
  position: fixed;
  inset: 0;
  margin: auto;
  padding: 0;
  border: 0;
  border-radius: 20px;
  background: #ffffff;
  box-shadow: 0 25px 60px -15px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(0, 0, 0, 0.05);
  width: min(520px, calc(100vw - 32px));
  max-height: calc(100dvh - 48px);
  overflow: hidden;
  box-sizing: border-box;
 }
 .pic-dialog::backdrop {
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(5px);
  animation: modalFadeInBackdrop 0.2s cubic-bezier(0.16, 1, 0.3, 1) forwards;
 }
 .pic-dialog[open] {
  animation: modalPopIn 0.22s cubic-bezier(0.16, 1, 0.3, 1) forwards;
 }
 @keyframes modalFadeInBackdrop {
  from { opacity: 0; }
  to { opacity: 1; }
 }
 @keyframes modalPopIn {
  from {
   opacity: 0;
   transform: scale(0.95) translateY(8px);
  }
  to {
   opacity: 1;
   transform: scale(1) translateY(0);
  }
 }

 .dialog-content {
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 48px);
  overflow-y: auto;
  padding: 24px;
  gap: 20px;
  box-sizing: border-box;
 }

 .dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--his-border, #dedede);
  padding-bottom: 16px;
 }
 .dialog-title-area {
  display: flex;
  flex-direction: column;
  gap: 4px;
 }
 .dialog-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--his-primary, #007a41);
 }
 .dialog-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--his-text, #171717);
  letter-spacing: -0.02em;
 }
 .dialog-job-name {
  margin: 2px 0 0;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--his-muted, #595959);
  line-height: 1.4;
  word-break: normal;
 }
 .dialog-close-btn {
  min-width: unset !important;
  min-height: unset !important;
  width: 34px !important;
  height: 34px !important;
  border-radius: 50%;
  border: 1px solid var(--his-border, #dedede);
  background: var(--his-surface, #f7f9f7);
  color: var(--his-muted, #595959);
  font-size: 1.25rem;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0 !important;
  transition: all 160ms ease;
 }
 .dialog-close-btn:hover:not(:disabled) {
  background: var(--his-danger-soft, #fff0f0);
  border-color: var(--his-danger-border, #ffc9ca);
  color: var(--his-danger, #c52327);
  transform: rotate(90deg);
 }

 .dialog-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 0.85rem;
  font-weight: 600;
 }
 .dialog-alert.error {
  background: var(--his-danger-soft, #fff0f0);
  color: var(--his-danger, #c52327);
  border: 1px solid var(--his-danger-border, #ffc9ca);
 }
 .dialog-alert.success {
  background: var(--his-primary-soft, #e8f8ef);
  color: var(--his-primary, #007a41);
  border: 1px solid #bbf7d0;
 }

 .dialog-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
 }
 .section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
 }
 .section-title {
  font-size: 0.82rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--his-muted, #595959);
 }
 .count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--his-surface, #f7f9f7);
  border: 1px solid var(--his-border, #dedede);
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--his-muted, #595959);
 }

 .assigned-pic-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 180px;
  overflow-y: auto;
  padding-right: 4px;
 }
 .assigned-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 12px;
  background: var(--his-surface, #f7f9f7);
  border: 1px solid var(--his-border, #dedede);
  transition: all 140ms ease;
 }
 .assigned-card:hover {
  border-color: #bbf7d0;
  background: #ffffff;
 }
 .assigned-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
 }
 .avatar-large {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--his-primary, #007a41), #00a859);
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  font-weight: 700;
  flex-shrink: 0;
 }
 .candidate-avatar {
  background: linear-gradient(135deg, #0284c7, #38bdf8);
 }
 .assigned-names, .candidate-names {
  display: flex;
  flex-direction: column;
  min-width: 0;
 }
 .assigned-fullname, .candidate-name {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--his-text, #171717);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
 }
 .assigned-meta, .candidate-username {
  font-size: 0.75rem;
  color: var(--his-muted, #595959);
  display: flex;
  align-items: center;
  gap: 6px;
 }
 .inactive-tag {
  color: var(--his-danger, #c52327);
  font-weight: 600;
 }
 .btn-remove-pic {
  min-width: unset !important;
  min-height: unset !important;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px !important;
  border-radius: 8px;
  border: 1px solid var(--his-danger-border, #ffc9ca);
  background: var(--his-danger-soft, #fff0f0);
  color: var(--his-danger, #c52327);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 140ms ease;
  flex-shrink: 0;
 }
 .btn-remove-pic:hover:not(:disabled) {
  background: var(--his-danger, #c52327);
  color: #ffffff;
 }

 .search-box {
  position: relative;
  display: flex;
  align-items: center;
 }
 .search-icon {
  position: absolute;
  left: 12px;
  color: var(--his-muted, #595959);
  pointer-events: none;
 }
 .search-box input {
  width: 100%;
  padding: 10px 36px 10px 36px !important;
  border: 1px solid var(--his-border, #dedede);
  border-radius: 10px;
  font-size: 0.88rem;
  transition: border-color 160ms ease, box-shadow 160ms ease;
 }
 .search-box input:focus {
  border-color: var(--his-primary, #007a41);
  box-shadow: 0 0 0 3px rgba(0, 122, 65, 0.12);
  outline: none;
 }
 .btn-clear-search {
  position: absolute;
  right: 10px;
  min-width: unset !important;
  min-height: unset !important;
  width: 20px !important;
  height: 20px !important;
  padding: 0 !important;
  border: 0;
  background: #e5e7eb;
  color: #4b5563;
  border-radius: 50%;
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
 }

 .candidates-area {
  display: flex;
  flex-direction: column;
  gap: 8px;
 }
 .candidates-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 200px;
  overflow-y: auto;
  padding-right: 4px;
 }
 .candidate-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 12px;
  background: #ffffff;
  border: 1px solid var(--his-border, #dedede);
  transition: all 140ms ease;
 }
 .candidate-row:hover {
  border-color: var(--his-primary, #007a41);
  background: var(--his-surface, #f7f9f7);
 }
 .candidate-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
 }
 .btn-add-candidate {
  min-width: unset !important;
  min-height: unset !important;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px !important;
  border-radius: 8px;
  border: 1px solid var(--his-primary, #007a41);
  background: var(--his-primary-soft, #e8f8ef);
  color: var(--his-primary, #007a41);
  font-size: 0.78rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 140ms ease;
  flex-shrink: 0;
 }
 .btn-add-candidate:hover:not(:disabled) {
  background: var(--his-primary, #007a41);
  color: #ffffff;
 }

 .empty-state {
  margin: 4px 0;
  font-size: 0.82rem;
  color: var(--his-muted, #595959);
  text-align: center;
  padding: 14px;
  background: var(--his-surface, #f7f9f7);
  border-radius: 10px;
 }
 .search-hint {
  font-size: 0.72rem;
  color: #8c8c8c;
 }
 .loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  font-size: 0.85rem;
  color: var(--his-muted, #595959);
 }
 .spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--his-border, #dedede);
  border-top-color: var(--his-primary, #007a41);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
 }
 @keyframes spin {
  to { transform: rotate(360deg); }
 }

 .dialog-footer {
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid var(--his-border, #dedede);
  padding-top: 14px;
 }
 .btn-done {
  min-width: unset !important;
  min-height: unset !important;
  padding: 9px 20px !important;
  border-radius: 10px;
  border: 0;
  background: var(--his-primary, #007a41);
  color: #ffffff;
  font-size: 0.88rem;
  font-weight: 700;
  cursor: pointer;
  transition: background-color 160ms ease, transform 160ms ease;
 }
 .btn-done:hover:not(:disabled) {
  background: var(--his-primary-dark, #005c31);
  transform: translateY(-1px);
 }
</style>
