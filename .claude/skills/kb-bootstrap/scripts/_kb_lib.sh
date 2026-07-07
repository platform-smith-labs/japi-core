# _kb_lib.sh — shared helpers for the kb-* scripts. SOURCE this, don't execute.
# Pure bash + awk (no python/PyYAML, nothing a target repo must install — NFR-2/NFR-5).
# KB frontmatter = the block between the first two '---' fences at the very top of a file.

# Print the frontmatter block (without the fences). Empty if the file has no leading frontmatter.
kb_fm_block() {
  awk 'NR==1 && $0 !~ /^---[[:space:]]*$/ {exit}
       /^---[[:space:]]*$/ {d++; if(d==1) next; if(d==2) exit}
       d==1 {print}' "$1"
}

# Print the body (everything after the 2nd fence), preserving any '---' inside it.
kb_body() {
  awk 'BEGIN{d=0}
       d>=2 {print; next}
       /^---[[:space:]]*$/ {d++; next}
       {next}' "$1"
}

# Print a scalar frontmatter value (quotes stripped). Empty if absent.
kb_fm_scalar() {
  kb_fm_block "$1" | awk -v k="$2" '
    $0 ~ "^"k":" { sub("^"k":[[:space:]]*",""); gsub(/^["'\'']+|["'\'']+$/,""); print; exit }'
}

# Print the number of items in a list frontmatter field (inline [a,b] or block "- item" form).
kb_fm_list_count() {
  kb_fm_block "$1" | awk -v k="$2" '
    BEGIN{state=0; c=0; result=""}
    result!="" {next}
    state==1 {
      if ($0 ~ /^[[:space:]]+-[[:space:]]*[^[:space:]]/) {c++; next}
      if ($0 ~ /^[^[:space:]]/) {result=c}
    }
    state==0 && $0 ~ "^"k":" {
      r=$0; sub("^"k":[[:space:]]*","",r)
      if (r ~ /^\[/) { gsub(/[][[:space:]]/,"",r); result=(r==""?0:split(r,a,",")) }
      else if (r=="") {state=1}
      else {result=0}
    }
    END{ if(result=="") result=c; print result+0 }'
}

# Print the items of a list frontmatter field, one per line (handles inline [a,b] and block "- item").
kb_fm_list_items() {
  kb_fm_block "$1" | awk -v k="$2" '
    BEGIN{state=0; done=0}
    done {next}
    state==1 {
      if ($0 ~ /^[[:space:]]+-[[:space:]]*[^[:space:]]/) { sub(/^[[:space:]]+-[[:space:]]*/,""); gsub(/^["'\'']+|["'\'']+$/,""); print; next }
      if ($0 ~ /^[^[:space:]]/) {done=1}
    }
    state==0 && $0 ~ "^"k":" {
      r=$0; sub("^"k":[[:space:]]*","",r)
      if (r ~ /^\[/) { gsub(/[][]/,"",r); n=split(r,a,","); for(i=1;i<=n;i++){gsub(/^[[:space:]"'\'']+|[[:space:]"'\'']+$/,"",a[i]); if(a[i]!="")print a[i]} done=1 }
      else if (r=="") {state=1}
      else {done=1}
    }'
}

# True if the file begins with a frontmatter block.
kb_has_fm() { [ -n "$(kb_fm_block "$1")" ]; }
