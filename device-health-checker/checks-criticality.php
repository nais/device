<?php declare(strict_types=1);
namespace Nais\Device;

/**
 * Criticality levels of all checks. The numeric keys are the check IDs from Kolide.
 *
 * List all checks by running the kolide:list-checks command
 */
return [
    1     => Criticality::CRIT,   // macOS - Bluetooth Sharing Enable
    2     => Criticality::CRIT,   // macOS - Disc Sharing Enabled
    3     => Criticality::HIGH,   // Unencrypted SSH Keys
    4     => Criticality::CRIT,   // Evil Chrome Extension - TouchVPN
    5     => Criticality::CRIT,   // Evil Chrome Extension - StartNewSearch
    6     => Criticality::CRIT,   // Evil Chrome Extension - Searchmanager
    7     => Criticality::HIGH,   // macOS - Find My Mac Disabled
    8     => Criticality::LOW,    // macOS - Finder File Extensions Hidden
    9     => Criticality::MED,    // Windows - Explorer Show All File Extensions Disabled
    10    => Criticality::CRIT,   // macOS - File Sharing Enabled
    11    => Criticality::HIGH,   // macOS - Firewall Disabled
    12    => Criticality::CRIT,   // Windows - Firewall Disabled
    13    => Criticality::CRIT,   // macOS - Gatekeeper Disabled
    14    => Criticality::CRIT,   // macOS - Internet Sharing Enabled
    15    => Criticality::CRIT,   // Malware - Adware Doctor (Files)
    16    => Criticality::CRIT,   // Malware - Adware Doctor (App)
    17    => Criticality::CRIT,   // Malware - Dr. Unarchiver
    18    => Criticality::CRIT,   // Malware - Dr. Antivirus
    19    => Criticality::CRIT,   // Malware - Dr. No Sleep
    20    => Criticality::CRIT,   // Malware - Dr. Cleaner
    21    => Criticality::CRIT,   // Malware - WireLurker
    22    => Criticality::CRIT,   // Malware - OSX/Leverage.A (launchd)
    23    => Criticality::CRIT,   // Malware - OSX/Leverage.A (Files)
    24    => Criticality::CRIT,   // Malware - Tibet.D
    25    => Criticality::CRIT,   // Malware - DevilRobber
    26    => Criticality::CRIT,   // Sudo Does Not Require Password
    27    => Criticality::MED,    // macOS - GitHub 2FA Codes stored in Plain-Text
    28    => Criticality::MED,    // macOS - GSuite 2FA Codes stored in Plain-Text
    29    => Criticality::MED,    // macOS - 1Password Emergency Kits stored in Plain-Text
    30    => Criticality::CRIT,   // macOS - Printer Sharing Enabled
    31    => Criticality::CRIT,   // macOS - FileVault2 Primary Disk Encryption Not Enabled
    32    => Criticality::CRIT,   // Linux - Primary Disk Encryption Not Enabled
    33    => Criticality::CRIT,   // Windows - Bitlocker Primary Disk Encryption Not Enabled
    34    => Criticality::CRIT,   // macOS - Remote Apple Events Enabled
    35    => Criticality::CRIT,   // macOS - Remote Login (SSH) Enabled
    36    => Criticality::CRIT,   // macOS - Remote Management Enabled
    37    => Criticality::MED,    // macOS - Terminal.app Secure Keyboard Entry Disabled
    38    => Criticality::MED,    // macOS - iTerm2.app Secure Keyboard Entry Disabled
    39    => Criticality::CRIT,   // macOS - System Integrity Protection (SIP) Disabled
    40    => Criticality::CRIT,   // Windows - User Account Control (UAC) Disabled
    41    => Criticality::CRIT,   // Vulnerability - Insecure Zoom Video Conference Server
    15384 => Criticality::IGNORE, // old_customer_data_export
    15804 => Criticality::IGNORE, // MacBook - Battery Unhealthy
    15805 => Criticality::IGNORE, // macOS - Primary Disk Almost Full
    15806 => Criticality::IGNORE, // Linux - Primary Disk Almost Full
    15807 => Criticality::IGNORE, // Windows - Primary Disk Almost Full
    27680 => Criticality::CRIT,   // macOS - Operating System Important Updates Missing
    29818 => Criticality::CRIT,   // Vulnerability - iTerm2 (CVE-2019-9535)
    47076 => Criticality::MED,    // Windows - Ransomware Protection (Controlled Folder Access) Disabled
    49356 => Criticality::CRIT,   // Windows - Screen Lock Disabled
    50322 => Criticality::CRIT,   // Windows - No Antivirus Products Configured
    53542 => Criticality::CRIT,   // Vulnerability - Windows CryptoAPI (CVE-2020-0601)
    75184 => Criticality::MED,    // GitHub 2FA Codes stored in Plain-Text
    75185 => Criticality::MED,    // GitHub 2FA Codes stored in Plain-Text
    75186 => Criticality::MED,    // GSuite 2FA Codes stored in Plain-Text
    75187 => Criticality::MED,    // GSuite 2FA Codes stored in Plain-Text
    75188 => Criticality::MED,    // 1Password Emergency Kits stored in Plain-Text
    75189 => Criticality::MED,    // 1Password Emergency Kits stored in Plain-Text
];
