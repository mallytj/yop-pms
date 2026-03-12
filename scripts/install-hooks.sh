# Link the pre-commit hook (Formatting)
ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
chmod +x scripts/pre-commit.sh
chmod +x .git/hooks/pre-commit

# Link the pre-push hook (Full Audit)
ln -sf ../../scripts/pre-push.sh .git/hooks/pre-push
chmod +x scripts/pre-push.sh
chmod +x .git/hooks/pre-push

echo "✅ All Git hooks (pre-commit & pre-push) installed."