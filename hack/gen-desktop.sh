#!/bin/bash

if [ ! -d ".git" ]; then
    echo "Error: incompatible working directory, you need to run this command from the root of the project."
    exit 1
fi

TARGET_DIR="desktop/src/gen"
if [ -d "$TARGET_DIR" ]; then rm -Rf $TARGET_DIR; fi

echo "Generating types..."
cd desktop/src-tauri
cargo test --quiet &> /dev/null 2>&1
cd ../..
printf "Done\n"

echo "Copying generated types..."
mkdir -p desktop/src/gen
cp -r desktop/src-tauri/bindings/* desktop/src/gen 
rm -rf desktop/src-tauri/bindings

# Add index.ts file for easy import
touch desktop/src/gen/index.ts
for filename in desktop/src/gen/*.ts; do
    echo "export * from './$(basename ${filename} .ts)'" >> desktop/src/gen/index.ts
done
printf "Done\n"

echo "Formatting generated types..."
cd desktop
yarn prettier --write src/gen --log-level silent
cd ..
printf "Done\n"
