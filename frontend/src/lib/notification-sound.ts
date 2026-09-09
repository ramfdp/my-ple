// One audio instance per active notification connection. No historical sound queue.
export function notificationSound() {
 const audio = new Audio('/notification.wav');
 audio.preload = 'auto';
 audio.volume = 0.25;
 let unlocked = false, unlocking = false, stopped = false, lastPlayed = -Infinity;
 const unlock = async () => {
  if (unlocked || unlocking || stopped) return;
  unlocking = true;
  try {
   audio.muted = true;
   await audio.play();
   audio.pause(); audio.currentTime = 0;
   unlocked = !stopped;
  } catch { /* Autoplay may remain blocked; retry on the next interaction. */ }
  finally { audio.muted = false; unlocking = false; }
 };
 window.addEventListener('pointerdown', unlock);
 window.addEventListener('keydown', unlock);
 return {
  play() {
   if (!unlocked || stopped || Date.now()-lastPlayed < 800) return;
   lastPlayed = Date.now(); audio.currentTime = 0;
   try { void audio.play().catch(() => {}); } catch { /* Audio never blocks notifications. */ }
  },
  stop() { stopped = true; audio.pause(); window.removeEventListener('pointerdown', unlock); window.removeEventListener('keydown', unlock); }
 };
}
