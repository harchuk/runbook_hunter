# Mattermost incoming webhook setup

1. In Mattermost, create an incoming webhook integration in the target workspace/channel.
2. Copy the generated webhook URL.
3. Put the URL into Helm Secret values or set it via UI overrides.

Security notes:
- Treat the webhook URL as a credential.
- Never log the webhook URL.
- Rotate webhook URL if exposed.
