<script lang="ts">
 import {page} from '$app/state';
 import WorkspaceView from '$lib/components/workspace/WorkspaceView.svelte';
</script>
<WorkspaceView id={page.params.id??''}/>
