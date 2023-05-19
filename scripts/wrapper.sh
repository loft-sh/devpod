tmpfile=$(mktemp)

cat > $tmpfile <<'EOL'
{{ .Script }}
EOL

chmod +x $tmpfile
$tmpfile
rm $tmpfile
