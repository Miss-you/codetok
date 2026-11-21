# ENG-013 final implementation

`final_impl_v1.md` with two changes found during the local rehearsal:

1. The change agent tried a shell `grep`, the policy denied it, and `acpx` exited 5 ("all permission requests denied"). The prompt now says the agent has no shell and must use its own read/search tools, and `run_agent` treats exit 5 as a normal end when the turn ended with `[done] end_turn`; the checks then judge the edits (an empty diff abandons the candidate).
2. The label check runs only in PR mode, so `--no-pr` rehearsals work before the label exists.

No OpenSpec change (`Change=-`): this is automation tooling and does not change CLI behavior.
