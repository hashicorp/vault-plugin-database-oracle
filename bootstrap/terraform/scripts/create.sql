-- This directive is essential for automation. It ensures that if any
-- SQL statement fails, the sqlplus client will exit immediately with
-- the Oracle error code, causing shell scripts with 'set e' to fail.
WHENEVER SQLERROR EXIT SQL.SQLCODE;

-- Use DEFINE to assign positional parameters to named variables for clarity.
DEFINE vault_admin          = '&1'
DEFINE vault_admin_password = '&2'
DEFINE num_static_users     = '&3'


-- Section 1: Clean Up and Create the Root User for the Vault Oracle Plugin
-- ---------------------------------------------------------------------
PROMPT Killing any lingering sessions for VAULT_ADMIN...
BEGIN
  FOR I IN (SELECT SID, SERIAL# FROM V$SESSION WHERE USERNAME = UPPER('&vault_admin')) LOOP
    EXECUTE IMMEDIATE 'ALTER SYSTEM KILL SESSION ''' || I.SID || ',' || I.SERIAL# || ''' IMMEDIATE';
  END LOOP;
  DBMS_LOCK.SLEEP(2);
EXCEPTION
  WHEN OTHERS THEN
    NULL;
END;
/

PROMPT Cleaning up previous VAULT_ADMIN user if it exists...
BEGIN
  EXECUTE IMMEDIATE 'DROP USER "&vault_admin" CASCADE';
EXCEPTION
  WHEN OTHERS THEN
    IF SQLCODE != -1918 THEN
      RAISE;
    END IF;
END;
/

PROMPT Creating Vault admin user '&vault_admin'...
CREATE USER "&vault_admin" IDENTIFIED BY "&vault_admin_password";

PROMPT Granting necessary privileges to '&vault_admin'...
GRANT CREATE SESSION TO "&vault_admin";
GRANT CREATE USER, ALTER USER, DROP USER TO "&vault_admin";
GRANT CREATE ROLE, DROP ANY ROLE TO "&vault_admin";
GRANT CONNECT TO "&vault_admin" WITH ADMIN OPTION;
GRANT UNLIMITED TABLESPACE TO "&vault_admin";
GRANT SELECT_CATALOG_ROLE TO "&vault_admin";


-- Section 2: Programmatically Create N Static Users and Roles
-- ---------------------------------------------------------------------
PROMPT
PROMPT --- PREPARING TO CREATE &num_static_users STATIC USERS AND ROLES ---
PROMPT
DECLARE
  v_user_name VARCHAR2(30);
  v_role_name VARCHAR2(30);
  v_password  VARCHAR2(40);
BEGIN
  DBMS_OUTPUT.PUT_LINE('--- Starting creation of ' || &num_static_users || ' static user(s) and role(s) ---');
  DBMS_OUTPUT.PUT_LINE('------------------------------------------------------------');

  -- Loop from 0 to N to match the Vault configuration script
  FOR i IN 0..&num_static_users LOOP
    v_user_name := 'STATIC_USER_' || i;
    v_role_name := 'APP_ROLE_' || i;

    -- Step A: Kill any lingering sessions for the static user.
    BEGIN
      FOR S IN (SELECT SID, SERIAL# FROM V$SESSION WHERE USERNAME = v_user_name) LOOP
        EXECUTE IMMEDIATE 'ALTER SYSTEM KILL SESSION ''' || S.SID || ',' || S.SERIAL# || ''' IMMEDIATE';
      END LOOP;
      DBMS_LOCK.SLEEP(2);
    EXCEPTION
      WHEN OTHERS THEN NULL; -- Ignore errors
    END;

    -- Step B: Drop the user.
    BEGIN
      EXECUTE IMMEDIATE 'DROP USER "' || v_user_name || '" CASCADE';
    EXCEPTION
      WHEN OTHERS THEN IF SQLCODE != -1918 THEN RAISE; END IF; -- Ignore "does not exist"
    END;
    -- Step C: Drop the role.
    BEGIN
      EXECUTE IMMEDIATE 'DROP ROLE "' || v_role_name || '"';
    EXCEPTION
      WHEN OTHERS THEN IF SQLCODE != -1919 THEN RAISE; END IF;
    END;

    v_password  := DBMS_RANDOM.STRING('x', 25) || 'a1!';
    EXECUTE IMMEDIATE 'CREATE USER "' || v_user_name || '" IDENTIFIED BY "' || v_password || '"';
    EXECUTE IMMEDIATE 'ALTER USER "' || v_user_name || '" ACCOUNT LOCK';
    EXECUTE IMMEDIATE 'CREATE ROLE "' || v_role_name || '"';
    EXECUTE IMMEDIATE 'GRANT CREATE SESSION TO "' || v_role_name || '"';
    EXECUTE IMMEDIATE 'GRANT "' || v_role_name || '" TO "&vault_admin" WITH ADMIN OPTION';

    DBMS_OUTPUT.PUT_LINE('Cleaned and Created: ' || RPAD(v_user_name, 15) || ' | Role: ' || RPAD(v_role_name, 15));
  END LOOP;

  DBMS_OUTPUT.PUT_LINE('------------------------------------------------------------');
  DBMS_OUTPUT.PUT_LINE('--- Finished creating static users and roles ---');
END;
/

PROMPT Script finished successfully.
COMMIT;
EXIT;

