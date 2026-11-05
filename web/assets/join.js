// Self-registration from an invite link: /join/<token>
const token = location.pathname.split("/").pop();
const form = document.getElementById("join-form");
const invalid = document.getElementById("invalid");

async function check() {
  const res = await fetch(`/api/invites/${encodeURIComponent(token)}`);
  if (res.ok) {
    form.hidden = false;
    document.getElementById("name").focus();
    return;
  }
  const data = await res.json().catch(() => ({}));
  invalid.textContent = data.error || "This invite isn’t valid.";
  invalid.hidden = false;
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const error = document.getElementById("error");
  const button = form.querySelector("button");
  error.hidden = true;
  button.disabled = true;
  try {
    const res = await fetch("/api/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        token,
        name: document.getElementById("name").value,
        team: document.getElementById("team").value,
        consent: document.getElementById("consent").checked,
      }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || "Registration failed.");
    document.getElementById("join").hidden = true;
    document.getElementById("done").hidden = false;
    document.getElementById("done-title").textContent = `You’re all set, ${data.name.split(" ")[0]}`;
  } catch (e) {
    error.textContent = e.message;
    error.hidden = false;
    button.disabled = false;
  }
});

check();
