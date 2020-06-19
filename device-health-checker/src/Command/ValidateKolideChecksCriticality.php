<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\Criticality;
use Nais\Device\KolideApiClient;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;
use RuntimeException;

class ValidateKolideChecksCriticality extends BaseCommand {
    /** @var string */
    protected static $defaultName = 'kolide:validate-checks';

    protected function configure() : void {
        $this
            ->setDescription('Validate Kolide checks criticality levels')
            ->setHelp('Make sure we have set criticality levels for all Kolide checks connected to our account')
            ->addOption('kolide-api-token', 't', InputOption::VALUE_REQUIRED, 'Token used with the Kolide API');
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
        array_multisort(array_column($checks, 'id'), SORT_ASC, $checks);
        $incompleteChecks = [];

        foreach ($checks as $check) {
            foreach ($check['tags'] as $tag) {
                if (Criticality::isValidTag($tag)) {
                    continue 2;
                }
            }

            $incompleteChecks[] = $check;
        }

        if (!empty($incompleteChecks)) {
            $output->writeln('The following Kolide checks are missing a severity tag:');
            $output->writeln(array_map(
                fn(array $check) : string => sprintf(
                    '<info>%s</info> (ID: <info>%d</info>, https://k2.kolide.com/1401/checks/%2$d): %s',
                    $check['name'],
                    $check['id'],
                    $check['description']
                ),
                $incompleteChecks
            ));
            return 1;
        }

        $output->writeln('All checks have been configured');

        return 0;
    }
}