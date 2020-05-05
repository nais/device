<?php declare(strict_types=1);
namespace Nais;

// Return checks configuration with regards to criticality. Will be removed once Kolide gets this
// information embedded in the API responses.
return [
    // 32848 => '', // Primary Disk Unhealthy
    // 32837 => '', // File Extensions Not Visible To User
    32847 => Criticality::HIGH, // Primary Disk Unencrypted
    32856 => Criticality::HIGH, // Serious Vulnerability
    32849 => Criticality::HIGH, // Ransomware Protection: Controlled Folder Access is Disabled
    32832 => Criticality::LOW, // macOS Bluetooth Sharing Enabled
    // 32831 => '', // Battery Requires Service
    32835 => Criticality::HIGH, // Evil Browser Extension
    32843 => Criticality::HIGH, // Operating System Missing Important Update
    32855 => Criticality::HIGH, // Windows UAC Disabled
    32842 => Criticality::HIGH, // Critical Malware
    32845 => Criticality::MED, // Plain-Text Recovery Codes on Disk
    32854 => Criticality::HIGH, // macOS System Integrity Protection Disabled
    32852 => Criticality::HIGH, // macOS Remote Management Enabled
    32851 => Criticality::HIGH, // macOS Remote Login (SSH) Enabled
    32838 => Criticality::LOW, // macOS File Sharing Enabled
    32858 => Criticality::MED, // Windows Screen Lock Disabled
    32844 => Criticality::HIGH, // Using Sudo Does Not Require Password
    32850 => Criticality::HIGH, // macOS Remote Apple Events Enabled
    32846 => Criticality::LOW, // macOS Printer Sharing Enabled
    32834 => Criticality::MED, // Unencrypted SSH Key
    32839 => Criticality::MED, // Firewall Disabled
    32840 => Criticality::HIGH, // macOS Gatekeeper Disabled
    32836 => Criticality::HIGH, // Find My Mac Disabled
    32853 => Criticality::HIGH, // macOS Secure Keyboard Entry Disabled
    32857 => Criticality::HIGH, // No Windows Antivirus Software Configured
    32841 => Criticality::LOW, // macOS Internet Sharing Enabled
];