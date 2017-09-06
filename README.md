# Process to delete files

***

Process that performs a deletion of files according to the parameters defined in the file 'config.json'.

The parameters are:
* _directories_: All the directories that are required to be deleted, separated by '|'.
* _extensions_: All the extensions that are required to be deleted, separated by '|'.
* _pattern_: Regular expression to  match the file name to be deleted.
* _days_of_expiration_: Days of antique to the file to be deleted.
* _search_in_subdirectories_: Search in the subdirectories to directories defined on the parameter _'directories'_.