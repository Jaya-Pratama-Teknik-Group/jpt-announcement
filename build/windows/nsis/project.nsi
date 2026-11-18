Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them.
##
## For development first build to populate "wails_tools.nsh":
##   > wails3 task windows:package
## Then call makensis manually:
##   AMD64:  makensis -DARG_WAILS_AMD64_BINARY=..\..\..\bin\JPTAnnounce.exe project.nsi
##   ARM64:  makensis -DARG_WAILS_ARM64_BINARY=..\..\..\bin\JPTAnnounce.exe project.nsi
####

## Include the wails tools
!include "wails_tools.nsh"

# --- Version info embedded in the installer PE header ---
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Setup"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI2.nsh"

# --- Installer / Uninstaller icons ---
!define MUI_ICON   "..\icon.ico"
!define MUI_UNICON "..\icon.ico"

# --- Finish page options ---
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING
!define MUI_ABORTWARNING_TEXT "Apakah Anda yakin ingin membatalkan instalasi ${INFO_PRODUCTNAME}?"

!define MUI_FINISHPAGE_RUN          "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT     "Jalankan ${INFO_PRODUCTNAME} setelah selesai"

# --- Welcome page text (Bahasa Indonesia) ---
!define MUI_WELCOMEPAGE_TITLE "Selamat Datang di Installer ${INFO_PRODUCTNAME}"
!define MUI_WELCOMEPAGE_TEXT "Wizard ini akan memandu Anda menginstal ${INFO_PRODUCTNAME} ${INFO_PRODUCTVERSION}.$\r$\n$\r$\nAplikasi ini digunakan untuk menjadwalkan dan memutar pengumuman audio secara otomatis di komputer Windows.$\r$\n$\r$\nDisarankan menutup semua aplikasi lain sebelum melanjutkan."
!define MUI_FINISHPAGE_TITLE "Instalasi ${INFO_PRODUCTNAME} Selesai"
!define MUI_FINISHPAGE_TEXT "${INFO_PRODUCTNAME} telah berhasil diinstal.$\r$\n$\r$\nKlik Selesai untuk menutup wizard ini."

# --- Installer pages ---
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

# --- Uninstaller pages ---
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

# --- Languages (default = Indonesian, fallback English) ---
!insertmacro MUI_LANGUAGE "Indonesian"
!insertmacro MUI_LANGUAGE "English"

# --- Output settings ---
Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-setup.exe"

# Install to C:\Program Files\JPT\JPT Announce  (requires admin / UAC)
# To install to %LOCALAPPDATA% without UAC, pass -DWAILS_INSTALL_SCOPE=user
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif

ShowInstDetails show
ShowUninstDetails show

Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

# =====================================================================
# INSTALL SECTION
# =====================================================================
Section "Aplikasi Utama" SecMain
    !insertmacro wails.setShellContext

    # Install / update WebView2 runtime if not already present
    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    # Copy the application executable
    !insertmacro wails.files

    # Shortcut: Start Menu > Programs > JPT > JPT Announce
    CreateDirectory "$SMPROGRAMS\${INFO_COMPANYNAME}"
    CreateShortcut "$SMPROGRAMS\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}.lnk" \
        "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 \
        SW_SHOWNORMAL "" "Penjadwalan Pengumuman Audio Otomatis"

    # Shortcut: Desktop
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" \
        "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    # Write uninstaller + Add/Remove Programs entry
    !insertmacro wails.writeUninstaller
SectionEnd

# =====================================================================
# UNINSTALL SECTION
# =====================================================================
Section "Uninstall"
    !insertmacro wails.setShellContext

    # Remove application files
    RMDir /r $INSTDIR

    # Remove WebView2 temp data
    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    # Remove shortcuts
    Delete "$SMPROGRAMS\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}.lnk"
    RMDir  "$SMPROGRAMS\${INFO_COMPANYNAME}"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    # Remove autostart registry entry if the user had enabled it
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "JPTAnnounce"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
