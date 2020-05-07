<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\KolideApiClient;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;
use RuntimeException;

class ListChecks extends BaseCommand {
    /** @var string */
    protected static $defaultName = 'kolide:list-checks';

    protected function configure() : void {
        $this
            ->setDescription('List Kolide checks')
            ->setHelp('This command will list all checks that is currently assigned to our account on Kolide.')
            ->addOption('kolide-api-token', 't', InputOption::VALUE_REQUIRED, 'Some option');
    }

    protected function initialize(InputInterface $input, OutputInterface $output) : void {
        if (null !== $this->kolideApiClient) {
            return;
        }

        if (empty($input->getOption('kolide-api-token'))) {
            throw new RuntimeException('Specity a token for the Kolide API using -t/--kolide-api-token');
        }

        $this->setKolideApiClient(new KolideApiClient($input->getOption('kolide-api-token')));
    }

    protected function execute(InputInterface $input, OutputInterface $output) : int {
        $checks = $this->kolideApiClient->getAllChecks();

        if (empty($checks)) {
            $output->writeln('No checks exist');
            return 0;
        }

        $maxIdLength   = strlen((string) max(array_column($checks, 'id')));
        $maxUrlLength  = strlen('https://k2.kolide.com/1401/checks/') + $maxIdLength;
        $maxNameLength = max(array_map('strlen', array_column($checks, 'name')));

        $output->writeln([
            sprintf(
                '%s | %s | URL',
                str_pad('ID', $maxIdLength, ' '),
                str_pad('Name', $maxNameLength, ' ')
            ),
            sprintf(
                '%s | %s | %s',
                str_repeat('=', $maxIdLength),
                str_repeat('=', $maxNameLength),
                str_repeat('=', $maxUrlLength)
            )
        ]);

        foreach ($checks as $check) {
            $output->writeln(sprintf(
                '%s | %s | https://k2.kolide.com/1401/checks/%d',
                str_pad((string) $check['id'], $maxIdLength, ' '),
                str_pad($check['name'], $maxNameLength, ' '),
                $check['id']
            ));
        }

        return 0;
    }
}