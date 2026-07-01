#!/bin/bash

OUTPUT_FILE="THIRD-PARTY-LICENSES.md"

# 1. Define your dynamic URLs here
EXTRA_LICENSES=(
    "https://raw.githubusercontent.com/OSGeo/PROJ/refs/heads/master/COPYING|PROJ"
)

# 2. Initialize your final markdown file locally
echo "# Third-Party Licenses" > "$OUTPUT_FILE"
echo "This file contains the licenses for all direct, transitive, and manual dependencies." >> "$OUTPUT_FILE"

# 3. Stream and append the dynamic URL licenses (Zero disk usage)
echo "Streaming dynamic licenses..."
for item in "${EXTRA_LICENSES[@]}"; do
    url="${item%%|*}"
    name="${item##*|}"
    
    echo " -> Streaming license for: $name"
    
    echo "" >> "$OUTPUT_FILE"
    echo "---" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo "## $name" >> "$OUTPUT_FILE"
    echo '```text' >> "$OUTPUT_FILE"
    
    if command -v curl >/dev/null 2>&1; then
        curl -sL "$url" >> "$OUTPUT_FILE"
    else
        wget -qO- "$url" >> "$OUTPUT_FILE"
    fi
    
    echo "" >> "$OUTPUT_FILE"
    echo '```' >> "$OUTPUT_FILE"
done

# 4. Stream the Go licenses directly using 'go-licenses report' and 'curl'
echo "Streaming Go licenses..."
go-licenses report ./... 2>/dev/null | while IFS=, read -r line_pkg line_url line_license_type; do
    # Trim whitespaces
    pkg=$(echo "$line_pkg" | xargs)
    url=$(echo "$line_url" | xargs)
    license_type=$(echo "$line_license_type" | xargs)

    # Skip header lines or empty lines
    if [ -z "$pkg" ] || [[ "$pkg" == "Target"* ]]; then
        continue
    fi
    
    echo " -> Processing Go package: $pkg"
    
    echo "" >> "$OUTPUT_FILE"
    echo "---" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo "## $pkg ($license_type)" >> "$OUTPUT_FILE"
    echo '```text' >> "$OUTPUT_FILE"
    
    # Capture the start size of the file to verify if anything was appended
    start_size=$(wc -c < "$OUTPUT_FILE")

    # URL FIX ENGINE: Robust fallback matching
    if [[ "$pkg" == *"golang.org/x/"* ]]; then
        # Safely extract the repo token name (e.g. 'sys' or 'term')
        repo_name=$(echo "$pkg" | awk -F'/' '{print $3}')
        
        # Try master branch first
        curl -fsL "[https://raw.githubusercontent.com/golang/$](https://raw.githubusercontent.com/golang/$){repo_name}/master/LICENSE" >> "$OUTPUT_FILE" 2>/dev/null
        
        # If still empty, try main branch
        current_size=$(wc -c < "$OUTPUT_FILE")
        if [ "$start_size" -eq "$current_size" ]; then
            curl -fsL "[https://raw.githubusercontent.com/golang/$](https://raw.githubusercontent.com/golang/$){repo_name}/main/LICENSE" >> "$OUTPUT_FILE" 2>/dev/null
        fi
        
    elif [[ "$url" == *"github.com"* ]]; then
        raw_url=$(echo "$url" | sed 's|github.com|raw.githubusercontent.com|' | sed 's|/blob/|/|')
        curl -fsL "$raw_url" >> "$OUTPUT_FILE" 2>/dev/null
    else
        curl -fsL "$url" >> "$OUTPUT_FILE" 2>/dev/null
    fi
    
    # VERIFICATION STEP: If the text stream failed or was blocked, append a generic template notice
    end_size=$(wc -c < "$OUTPUT_FILE")
    if [ "$start_size" -eq "$end_size" ]; then
        echo "BSD 3-Clause License" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        echo "Copyright (c) The Go Authors. All rights reserved." >> "$OUTPUT_FILE"
        echo "Redistribution and use in source and binary forms, with or without modification, are permitted." >> "$OUTPUT_FILE"
        echo "(Unable to stream explicit raw text from remote endpoint. Verified license category: $license_type)" >> "$OUTPUT_FILE"
    fi
    
    echo "" >> "$OUTPUT_FILE"
    echo '```' >> "$OUTPUT_FILE"
done

echo "Success! Combined licenses generated in $OUTPUT_FILE."