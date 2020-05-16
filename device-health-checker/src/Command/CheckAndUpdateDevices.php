<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\ApiServerClient;
use Nais\Device\Criticality;
use Nais\Device\KolideApiClient;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;
use DateTime;
use RuntimeException;

class CheckAndUpdateDevices extends BaseCommand {
    /** @var string */
    protected static $defaultName = 'apiserver:update-devices';

    /** @var int */
    private $defaultCriticality = Criticality::MED;

    /** @var array */
    private $checksConfig = [
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
    ];

    protected function configure() : void {
        $this
            ->setDescription('Update health status of Nais devices')
            ->setHelp('This command will update the health status of all Nais devices based on data from the Kolide API.')
            ->addOption('kolide-api-token', 't', InputOption::VALUE_REQUIRED, 'Token used with the Kolide API')
            ->addOption('ignore-checks', 'i', InputOption::VALUE_IS_ARRAY|InputOption::VALUE_OPTIONAL, 'List of check IDs to ignore', [])
            ->addOption('apiserver-username', 'u', InputOption::VALUE_OPTIONAL, 'Username used for API server authentication (basic auth)', 'device-health-checker')
            ->addOption('apiserver-password', 'p', InputOption::VALUE_REQUIRED, 'Password used for API server authentication (basic auth)');
    }

    protected function initialize(InputInterface $input, OutputInterface $output) : void {
        if (null === $this->kolideApiClient && empty($input->getOption('kolide-api-token'))) {
            throw new RuntimeException('Specity a token for the Kolide API using -t/--kolide-api-token');
        } else if (null === $this->apiServerClient && empty($input->getOption('apiserver-username'))) {
            throw new RuntimeException('Specity a username for the API serveer using -u/--apiserver-username');
        } else if (null === $this->apiServerClient && empty($input->getOption('apiserver-password'))) {
            throw new RuntimeException('Specity a password for the API serveer using -p/--apiserver-password');
        }

        if (null === $this->kolideApiClient) {
            $this->setKolideApiClient(new KolideApiClient($input->getOption('kolide-api-token')));
        }

        if (null === $this->apiServerClient) {
            $this->setApiServerClient(new ApiServerClient($input->getOption('apiserver-username'), $input->getOption('apiserver-password')));
        }
    }

    protected function execute(InputInterface $input, OutputInterface $output) : int {
        // Force our own exception handler from here on
        $this->getApplication()->setCatchExceptions(false);

        $ignoreChecks = array_unique(array_map('intval', $input->getOption('ignore-checks')));
        $naisDevices = array_map(fn(array $device) : array => [
            'serial'    => $device['serial'],
            'platform'  => $device['platform'],
            'username'  => $device['username'],
            'isHealthy' => $device['isHealthy'],
        ], $this->apiServerClient->getDevices());
        $kolideDevices = $this->kolideApiClient->getAllDevices();
        $updatedNaisDevices = [];

        foreach ($naisDevices as $naisDevice) {
            $failingChecks     = [];
            $username          = $naisDevice['username'];
            $serial            = $naisDevice['serial'];
            $platform          = $naisDevice['platform'];
            $kolideDevice      = $this->identifyKolideDevice($username, $serial, $platform, $kolideDevices);

            if (null === $kolideDevice) {
                $this->log($output, 'Could not find matching Kolide device', $serial, $platform, $username);

                // Could not identify a single Kolide device for this Nais device
                $naisDevice['isHealthy'] = false;
                $updatedNaisDevices[] = $naisDevice;

                continue;
            }

            if ($kolideDevice['failure_count'] > $kolideDevice['resolved_failure_count']) {
                $failingChecks = $this->getFailingDeviceChecks($kolideDevice['id'], $ignoreChecks);
            }

            $isHealthy = 0 === count($failingChecks);

            if ($isHealthy && !$naisDevice['isHealthy']) {
                $this->log($output, 'No failing checks anymore, device is now healthy', $serial, $platform, $username);
            } else if (!$isHealthy && $naisDevice['isHealthy']) {
                $failingChecks = array_map(fn(array $check) : string => $check['title'], $failingChecks);

                $this->log(
                    $output,
                    sprintf('Device is no longer healthy because of the following failing check(s): %s', join(', ', $failingChecks)),
                    $serial, $platform, $username
                );
            }

            $naisDevice['isHealthy'] = $isHealthy;
            $updatedNaisDevices[] = $naisDevice;
        }

        if (empty($updatedNaisDevices)) {
            $this->log($output, 'No Nais devices to update :(');
            return 0;
        }

        $this->apiServerClient->updateDevices($updatedNaisDevices);
        $this->log($output, 'Sent updated Nais device configuration to API server');

        return 0;
    }

    /**
     * Identify a Kolide device for a given serial and platform
     *
     * Return the matching Kolide device. If multiple or no devices are found, return null.
     *
     * @param string $username
     * @param string $serial
     * @param string $platform
     * @param array $kolideDevices
     * @return ?array Returns null if no Kolide device matches
     */
    private function identifyKolideDevice(string $username, string $serial, string $platform, array $kolideDevices) : ?array {
        $devices = array_values(array_filter($kolideDevices, function(array $kolideDevice) use ($username, $serial, $platform) : bool {
            // Currently we only have darwin, windows or linux as possible platforms in the
            // apiserver, so if the Kolide device is not windows or darwin, assume linux.
            if (!in_array($kolideDevice['platform'], ['windows', 'darwin'])) {
                $kolideDevice['platform'] = 'linux';
            }

            return
                strtolower($username) === strtolower($kolideDevice['assigned_owner']['email'])
                && strtolower($serial) === strtolower($kolideDevice['serial'])
                && strtolower($platform) === strtolower($kolideDevice['platform']);
        }));

        if (empty($devices) || 1 < count($devices)) {
            return null;
        }

        return $devices[0];
    }

    /**
     * Check if a device is currently failing
     *
     * @param int $deviceId ID of the device at Kolide
     * @param array $ignoreChecks A list of check IDs to ignore
     * @return array
     */
    private function getFailingDeviceChecks(int $deviceId, array $ignoreChecks = []) : array {
        $failures = $this->kolideApiClient->getDeviceFailures($deviceId);
        $failingChecks = [];

        foreach ($failures as $failure) {
            if (null !== $failure['resolved_at']) {
                // Failure has been resolved
                continue;
            }

            $criticality = $this->checksConfig[$failure['check_id']] ?? $this->defaultCriticality;

            if (Criticality::IGNORE === $criticality || in_array($failure['check_id'], $ignoreChecks)) {
                continue;
            }

            $occurredAt = (new DateTime($failure['timestamp']))->getTimestamp();

            if (Criticality::CRIT === $criticality || ((time() - $occurredAt) > $criticality)) {
                $failingChecks[] = $failure;
            }
        }

        return $failingChecks;
    }

    /**
     * Output a log message in JSON
     *
     * @param OutputInterface $output
     * @param string $message
     * @param string $serial
     * @param string $platform
     * @param string $username
     * @return void
     */
    private function log(OutputInterface $output, string $message, string $serial = null, string $platform = null, string $username = null) : void {
        $output->writeln(json_encode(array_filter([
            'component' => 'device-health-checker',
            'system'    => 'nais-device',
            'message'   => $message,
            'serial'    => $serial,
            'platform'  => $platform,
            'username'  => $username,
            'level'     => 'info',
            'timestamp' => time(),
        ])));
    }
}