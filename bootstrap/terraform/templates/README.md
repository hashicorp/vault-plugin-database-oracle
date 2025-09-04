# Template File Variable Guide

This directory contains a Terraform template file. The '$' character has
special meaning and must be used carefully.

1. Terraform Variables: `${variable}`
   - Use for values passed in from your Terraform configuration.
   - Terraform replaces these *before* the script is created.
   - Example: ${tpl_vault_zip_file}

2. Shell Variables: $${variable}
   - Use for variables that the user-data script itself will set
     and use. The double dollar sign escapes it from Terraform.
   - Example: $${PRIVATE_IP}

3. Variables for Other Programs (like systemd): \$variable
   - Use a backslash to write a literal variable name (e.g., $MAINPID)
     into a configuration file. This prevents the user-data script's
     shell from trying to expand it.
   - Example: ExecReload=/bin/kill --signal HUP \$MAINPID

4. Special Shell PID Variable: $$
   - Use a double dollar sign with no variable name to get the
     Process ID (PID) of the script itself.
   - Example: echo "Script PID is $$"
--------------------------------------------------------------------

