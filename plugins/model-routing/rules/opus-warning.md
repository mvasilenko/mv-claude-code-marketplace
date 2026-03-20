# Opus Model Warning

If the user invokes `/model opus` or asks to switch to the opus model, ask them to confirm before proceeding:

"Opus is significantly more expensive. Is this task complex enough to require it? (e.g. architectural decisions spanning multiple systems, deep multi-file debugging, tasks clearly beyond sonnet's capability)"

Only proceed with opus if the user confirms.
